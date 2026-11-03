package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type RecentAuthentication interface {
	RequireRecent(context.Context, string) error
}

func (v firebaseVerifier) RequireRecent(ctx context.Context, raw string) error {
	token, err := v.client.VerifyIDTokenAndCheckRevoked(ctx, raw)
	if err != nil {
		return err
	}
	age := time.Since(time.Unix(token.AuthTime, 0))
	if age < -time.Minute || age > 5*time.Minute {
		return errors.New("recent authentication required")
	}
	return nil
}

type DeletionRepository interface {
	RequestDeletion(context.Context, string) error
	RunDeletions(context.Context)
}

func (m mongoUsers) RequestDeletion(ctx context.Context, user string) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		return nil, m.queueAccountDeletion(tx, user)
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}

// queueAccountDeletion must run inside the caller's transaction. App, web and
// ownership-reviewed support requests share the same cancellation/job invariants.
func (m mongoUsers) queueAccountDeletion(tx context.Context, user string) error {
	db := m.collection.Database()
	now := time.Now().UTC()
	var profile struct {
		UID string `bson:"firebase_uid"`
	}
	if err := m.collection.FindOne(tx, bson.M{"user_id": user}).Decode(&profile); err != nil {
		return err
	}
	if profile.UID == "" {
		return errors.New("missing identity")
	}
	if _, err := db.Collection("account_deletions").UpdateOne(tx, bson.M{"_id": user}, bson.M{"$setOnInsert": bson.M{"firebase_uid": profile.UID, "status": "pending", "created_at": now, "attempts": 0}}, options.UpdateOne().SetUpsert(true)); err != nil {
		return err
	}
	if _, err := m.collection.UpdateOne(tx, bson.M{"user_id": user}, bson.M{"$set": bson.M{"status": "deleting", "updated_at": now}, "$unset": bson.M{"fcm_token": ""}}); err != nil {
		return err
	}
	cursor, err := db.Collection("ride_groups").Find(tx, bson.M{"members.user_id": user, "status": bson.M{"$in": []string{"pending", "confirming", "active", "in_progress"}}})
	if err != nil {
		return err
	}
	defer cursor.Close(tx)
	var groups []GroupRecord
	if err = cursor.All(tx, &groups); err != nil {
		return err
	}
	for _, group := range groups {
		if _, err = db.Collection("ride_groups").UpdateOne(tx, bson.M{"group_id": group.ID}, bson.M{"$set": bson.M{"status": "cancelled", "updated_at": now}}); err != nil {
			return err
		}
		if _, err = m.rides().UpdateMany(tx, bson.M{"group_id": group.ID, "status": "matched"}, bson.M{"$set": bson.M{"status": "cancelled"}}); err != nil {
			return err
		}
		if _, err = db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "group_cancelled", "group_id": group.ID, "status": "pending", "attempts": 0, "created_at": now}); err != nil {
			return err
		}
	}
	if _, err = m.rides().UpdateMany(tx, bson.M{"user_id": user, "status": bson.M{"$in": []string{"pending", "matching"}}}, bson.M{"$set": bson.M{"status": "cancelled"}}); err != nil {
		return err
	}
	return nil
}

func (m mongoUsers) RunDeletions(ctx context.Context) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			work, cancel := context.WithTimeout(ctx, 60*time.Second)
			err := m.deleteNextAccount(work)
			m.recordWorker(ctx, "account_deletion", err)
			cancel()
		}
	}
}
func (m mongoUsers) deleteNextAccount(ctx context.Context) error {
	if m.identity == nil {
		return errors.New("identity deletion unavailable")
	}
	now := time.Now().UTC()
	jobs := m.collection.Database().Collection("account_deletions")
	lease := stableID(time.Now().String())
	var job struct {
		User     string `bson:"_id"`
		UID      string `bson:"firebase_uid"`
		Attempts int    `bson:"attempts"`
	}
	query := bson.M{"status": bson.M{"$in": []string{"pending", "processing"}}, "$and": bson.A{
		bson.M{"$or": bson.A{bson.M{"lease_until": bson.M{"$lte": now}}, bson.M{"lease_until": bson.M{"$exists": false}}}},
		bson.M{"$or": bson.A{bson.M{"next_attempt_at": bson.M{"$lte": now}}, bson.M{"next_attempt_at": bson.M{"$exists": false}}}},
	}}
	err := jobs.FindOneAndUpdate(ctx, query, bson.M{"$set": bson.M{"status": "processing", "lease_token": lease, "lease_until": now.Add(90 * time.Second)}, "$inc": bson.M{"attempts": 1}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&job)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	if err != nil {
		return err
	}
	cleanupErr := m.identity.DeleteUser(ctx, job.UID)
	if auth.IsUserNotFound(cleanupErr) {
		cleanupErr = nil
	}
	if cleanupErr == nil {
		cleanupErr = m.eraseAccountData(ctx, job.User, job.UID)
	}
	filter := bson.M{"_id": job.User, "lease_token": lease}
	if cleanupErr != nil {
		_, err = jobs.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"status": "pending", "next_attempt_at": time.Now().Add(time.Duration(min(3600, 30*(1<<min(job.Attempts, 7)))) * time.Second), "last_error": "Account cleanup requires retry"}, "$unset": bson.M{"lease_token": "", "lease_until": ""}})
		if err != nil {
			return err
		}
		return errors.New("account cleanup scheduled for retry")
	}
	_, err = jobs.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"status": "completed", "completed_at": time.Now().UTC(), "expires_at": time.Now().Add(7 * 24 * time.Hour)}, "$unset": bson.M{"firebase_uid": "", "lease_token": "", "lease_until": "", "last_error": "", "support_case_id": "", "reviewed_by": "", "reviewed_at": ""}})
	return err
}
func (m mongoUsers) eraseAccountData(ctx context.Context, user, firebaseUID string) error {
	db := m.collection.Database()
	cursor, err := db.Collection("ride_groups").Find(ctx, bson.M{"$or": bson.A{bson.M{"members.user_id": user}, bson.M{"origin_user_id": user}}})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	var groups []GroupRecord
	if err = cursor.All(ctx, &groups); err != nil {
		return err
	}
	groupIDs := []string{}
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	if len(groupIDs) > 0 {
		if _, err = db.Collection("reports").UpdateMany(ctx, bson.M{"group_id": bson.M{"$in": groupIDs}}, bson.M{"$set": bson.M{"chat_evidence": bson.A{}, "chat_excerpt": nil}}); err != nil {
			return err
		}
	}
	for _, name := range []string{"ride_requests", "notifications", "support_tickets", "user_logs", "user_avatars", "push_token_claims", "recurring_lifts", "web_push_tokens"} {
		if _, err := db.Collection(name).DeleteMany(ctx, bson.M{"user_id": user}); err != nil {
			return err
		}
	}
	// A deleted moderator can have authored messages in someone else's ticket.
	// Keep the other owner's thread, but remove this identity and its free text.
	if _, err := db.Collection("support_tickets").UpdateMany(ctx, bson.M{"messages.sender_id": user}, bson.M{"$set": bson.M{
		"messages.$[message].sender_id":   "deleted",
		"messages.$[message].sender_name": "Support",
		"messages.$[message].content":     "[Message removed after account deletion]",
	}}, options.UpdateMany().SetArrayFilters([]any{bson.M{"message.sender_id": user}})); err != nil {
		return err
	}
	if _, err := db.Collection("support_tickets").UpdateMany(ctx, bson.M{"reported_user_id": user}, bson.M{"$set": bson.M{
		"reported_user_id": nil, "subject": "Safety support request", "description": "[Removed after account deletion]", "messages": bson.A{},
	}}); err != nil {
		return err
	}
	if _, err := db.Collection("support_tickets").UpdateMany(ctx, bson.M{"resolved_by": user}, bson.M{"$set": bson.M{"resolved_by": nil}}); err != nil {
		return err
	}
	if _, err := db.Collection("reports").UpdateMany(ctx, bson.M{"chat_evidence.user_id": user}, bson.M{"$set": bson.M{"chat_evidence": bson.A{}, "chat_excerpt": nil}}); err != nil {
		return err
	}
	for _, name := range []string{"ratings", "user_blocks"} {
		fields := []string{"rater_user_id", "rated_user_id"}
		if name == "user_blocks" {
			fields = []string{"blocker_user_id", "blocked_user_id"}
		}
		if _, err := db.Collection(name).DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{fields[0]: user}, bson.M{fields[1]: user}}}); err != nil {
			return err
		}
	}
	if _, err := db.Collection("chat_messages").UpdateMany(ctx, bson.M{"user_id": user}, bson.M{"$set": bson.M{"user_id": "", "user_display_name": "Deleted account", "content": "[Message removed]"}, "$unset": bson.M{"dedupe_key": "", "client_message_id": ""}}); err != nil {
		return err
	}
	if _, err := db.Collection("ride_groups").UpdateMany(ctx, bson.M{"$or": bson.A{bson.M{"members.user_id": user}, bson.M{"origin_user_id": user}}}, bson.M{"$pull": bson.M{"members": bson.M{"user_id": user}}, "$set": bson.M{"route_summary": "Ride group", "pickup_summary": "Meeting point removed", "drop_summary": "Destination removed", "road_plan": nil, "route_search": nil, "reminder_schedule": nil}}); err != nil {
		return err
	}
	if _, err := db.Collection("reports").UpdateMany(ctx, bson.M{"$or": bson.A{bson.M{"reporter_user_id": user}, bson.M{"accused_user_id": user}}}, bson.M{"$set": bson.M{"note": "[Removed after account deletion]", "chat_evidence": bson.A{}, "chat_excerpt": nil, "resolution_note": "", "messages": bson.A{}}}); err != nil {
		return err
	}
	for _, field := range []string{"reporter_user_id", "accused_user_id", "resolved_by"} {
		if _, err := db.Collection("reports").UpdateMany(ctx, bson.M{field: user}, bson.M{"$set": bson.M{field: "deleted"}}); err != nil {
			return err
		}
	}
	// Remove actor-linked audit free text rather than assuming it contains no PII.
	if _, err := db.Collection("admin_logs").DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{"actor_id": user}, bson.M{"target_id": user}}}); err != nil {
		return err
	}
	if _, err := db.Collection("notification_outbox").DeleteMany(ctx, bson.M{"target_user_id": user}); err != nil {
		return err
	}
	if _, err := db.Collection("notification_outbox").UpdateMany(ctx, bson.M{"user_id": user}, bson.M{"$set": bson.M{"user_id": ""}}); err != nil {
		return err
	}
	receipt := "delivered." + stableID(user)
	if _, err := db.Collection("notification_outbox").UpdateMany(ctx, bson.M{receipt: bson.M{"$exists": true}}, bson.M{"$unset": bson.M{receipt: ""}}); err != nil {
		return err
	}
	if len(groupIDs) > 0 {
		if _, err := db.Collection("notifications").UpdateMany(ctx, bson.M{"data.group_id": bson.M{"$in": groupIDs}}, bson.M{
			"$set":   bson.M{"title": "Trip update", "body": "Open ORIX for current trip details."},
			"$unset": bson.M{"data.pickup_summary": "", "data.drop_summary": "", "data.route_summary": "", "data.road_plan": ""},
		}); err != nil {
			return err
		}
	}
	// Remove the final group-to-identity link only after every dependent group
	// redaction succeeds. Earlier retry attempts can still rediscover these groups.
	if _, err := db.Collection("ride_groups").UpdateMany(ctx, bson.M{"origin_user_id": user}, bson.M{"$unset": bson.M{"origin_user_id": ""}}); err != nil {
		return err
	}
	// A bootstrap marker must not retain the deleted Firebase identity or prevent
	// a future explicit bootstrap when no administrators remain.
	if _, err := db.Collection("app_config").DeleteOne(ctx, bson.M{"_id": "admin_bootstrap", "firebase_uid": firebaseUID}); err != nil {
		return err
	}
	_, err = m.collection.DeleteOne(ctx, bson.M{"user_id": user, "status": "deleting"})
	return err
}
func (a *AuthDependencies) deleteAccount(w http.ResponseWriter, r *http.Request, user User) {
	recent, ok := a.Verifier.(RecentAuthentication)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Recent sign-in verification unavailable"})
		return
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || recent.RequireRecent(r.Context(), parts[1]) != nil {
		writeJSON(w, 401, map[string]any{"detail": map[string]string{"error": "requires_recent_login", "message": "Sign in again before deleting your account"}})
		return
	}
	repository, ok := a.Users.(DeletionRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Account deletion unavailable"})
		return
	}
	if err := repository.RequestDeletion(r.Context(), user.ID); err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to request deletion. Please try again."})
		return
	}
	writeJSON(w, 202, map[string]any{"success": true, "status": "deleting", "message": "Account deletion requested. You will be signed out."})
}

// Deletion stays available to app-restricted accounts with a valid recent identity.
func (a *AuthDependencies) deleteAccountEndpoint(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.Verifier == nil || a.Users == nil {
		writeJSON(w, 503, map[string]string{"detail": "Account deletion unavailable"})
		return
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) > 16384 {
		writeJSON(w, 401, map[string]string{"detail": "Sign in again to delete your account"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	identity, err := a.Verifier.Verify(ctx, parts[1])
	if err != nil {
		writeJSON(w, 401, map[string]string{"detail": "Sign in again to delete your account"})
		return
	}
	user, _, err := a.Users.GetOrCreate(ctx, identity)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load your account"})
		return
	}
	a.deleteAccount(w, r.WithContext(ctx), user)
}
