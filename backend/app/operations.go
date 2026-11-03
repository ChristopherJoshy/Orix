package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type FailedDelivery struct {
	ID       any       `json:"event_id" bson:"_id"`
	Type     string    `json:"type" bson:"type"`
	Attempts int       `json:"attempts" bson:"attempts"`
	Created  time.Time `json:"created_at" bson:"created_at"`
}
type OperationsRepository interface {
	JobStatus(context.Context) (map[string]any, error)
	FailedDeliveries(context.Context) ([]FailedDelivery, error)
	RetryDelivery(context.Context, string, string) error
}

func (m mongoUsers) JobStatus(ctx context.Context) (map[string]any, error) {
	db := m.collection.Database()
	result := map[string]any{}
	for _, name := range []string{"notification_outbox", "account_deletions"} {
		counts := map[string]int64{}
		for _, state := range []string{"pending", "processing", "failed", "completed", "sent"} {
			count, err := db.Collection(name).CountDocuments(ctx, bson.M{"status": state})
			if err != nil {
				return nil, err
			}
			counts[state] = count
		}
		attention, err := db.Collection(name).CountDocuments(ctx, bson.M{"status": bson.M{"$in": []string{"pending", "processing", "failed"}}, "attempts": bson.M{"$gte": 8}})
		if err != nil {
			return nil, err
		}
		counts["needs_attention"] = attention
		result[name] = counts
	}
	cursor, err := db.Collection("worker_health").Find(ctx, bson.M{}, options.Find().SetLimit(20))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rows := []bson.M{}
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	result["workers"] = rows
	return result, nil
}
func (m mongoUsers) FailedDeliveries(ctx context.Context) ([]FailedDelivery, error) {
	cursor, err := m.collection.Database().Collection("notification_outbox").Find(ctx, bson.M{"status": "failed"}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(100))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rows := []FailedDelivery{}
	err = cursor.All(ctx, &rows)
	return rows, err
}
func (m mongoUsers) RetryDelivery(ctx context.Context, actor, eventID string) error {
	var id any = eventID
	if objectID, err := bson.ObjectIDFromHex(eventID); err == nil {
		id = objectID
	}
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		db := m.collection.Database()
		result, err := db.Collection("notification_outbox").UpdateOne(tx, bson.M{"_id": id, "status": "failed"}, bson.M{"$set": bson.M{"status": "pending", "attempts": 0}, "$unset": bson.M{"last_error": "", "lease_token": "", "lease_until": "", "next_attempt_at": ""}})
		if err != nil {
			return nil, err
		}
		if result.MatchedCount != 1 {
			return nil, ErrGroupUnavailable
		}
		_, err = WriteAudit(tx, db, bson.M{"actor_id": actor, "action": "retry_delivery", "target_id": eventID, "created_at": time.Now().UTC()})
		return nil, err
	})
	return err
}
func (a *AuthDependencies) operations(action string) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		repository, ok := a.Users.(OperationsRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Operations storage unavailable"})
			return
		}
		if action == "status" {
			state, err := repository.JobStatus(r.Context())
			if err != nil {
				writeJSON(w, 503, map[string]string{"detail": "Unable to load job status"})
				return
			}
			writeJSON(w, 200, state)
			return
		}
		if action == "failed" {
			rows, err := repository.FailedDeliveries(r.Context())
			if err != nil {
				writeJSON(w, 503, map[string]string{"detail": "Unable to load delivery failures"})
				return
			}
			writeJSON(w, 200, map[string]any{"events": rows})
			return
		}
		id := r.PathValue("event_id")
		if id == "" || len(id) > 256 {
			writeJSON(w, 422, map[string]string{"detail": "Invalid event ID"})
			return
		}
		if err := repository.RetryDelivery(r.Context(), user.ID, id); err != nil {
			if adminAccessDenied(w, err) {
				return
			}
			if errors.Is(err, ErrGroupUnavailable) {
				writeJSON(w, 409, map[string]string{"detail": "Only failed events can be retried"})
				return
			}
			writeJSON(w, 503, map[string]string{"detail": "Unable to retry event"})
			return
		}
		writeJSON(w, 200, map[string]bool{"success": true})
	}
}

// Admin responses deliberately use explicit projections. Operational access must
// not expose Firebase credentials, push tokens or arbitrary Mongo document fields.
type AdminGroupMember struct {
	ID       string `json:"user_id" bson:"user_id"`
	Ready    bool   `json:"readiness_confirmed" bson:"readiness_confirmed"`
	Here     bool   `json:"is_here" bson:"is_here"`
	Finished bool   `json:"finished" bson:"finished"`
}

type AdminGroupSummary struct {
	ID        string             `json:"group_id" bson:"group_id"`
	Status    string             `json:"status" bson:"status"`
	Mode      string             `json:"mode" bson:"mode"`
	Pickup    string             `json:"pickup_summary" bson:"pickup_summary"`
	Drop      string             `json:"drop_summary" bson:"drop_summary"`
	Departure time.Time          `json:"departure_start" bson:"departure_start"`
	End       time.Time          `json:"departure_end" bson:"departure_end"`
	Members   []AdminGroupMember `json:"members" bson:"members"`
}
type AdminAuditEntry struct {
	ID      bson.ObjectID `json:"log_id" bson:"_id"`
	Created time.Time     `json:"timestamp" bson:"created_at"`
	Actor   string        `json:"actor_id" bson:"actor_id"`
	Action  string        `json:"action" bson:"action"`
	Target  string        `json:"target_id" bson:"target_id"`
}
type AdminReadRepository interface {
	LookupAccount(context.Context, string, string) (User, error)
	AdminGroup(context.Context, string) (AdminGroupSummary, error)
	AuditEntries(context.Context, string) ([]AdminAuditEntry, error)
	AdminSummary(context.Context) (map[string]any, error)
}

func (m mongoUsers) LookupAccount(ctx context.Context, field, value string) (User, error) {
	var user User
	// Both field names come from server-side validation, never query JSON.
	if field != "user_id" && field != "email" {
		return user, ErrModerationConflict
	}
	cursor, err := m.collection.Find(ctx, bson.M{field: value}, options.Find().SetLimit(2))
	if err != nil {
		return user, err
	}
	defer cursor.Close(ctx)
	var users []User
	if err := cursor.All(ctx, &users); err != nil {
		return user, err
	}
	if len(users) == 0 {
		return user, mongo.ErrNoDocuments
	}
	// Legacy email duplicates must be disambiguated using the stable user ID.
	if len(users) != 1 {
		return user, ErrModerationConflict
	}
	return users[0], nil
}
func (m mongoUsers) AdminGroup(ctx context.Context, id string) (AdminGroupSummary, error) {
	var group AdminGroupSummary
	err := m.collection.Database().Collection("ride_groups").FindOne(ctx, bson.M{"group_id": id}, options.FindOne().SetProjection(bson.M{
		"group_id": 1, "status": 1, "mode": 1, "pickup_summary": 1, "drop_summary": 1,
		"departure_start": 1, "departure_end": 1, "members.user_id": 1,
		"members.readiness_confirmed": 1, "members.is_here": 1, "members.finished": 1,
	})).Decode(&group)
	if group.Members == nil {
		group.Members = []AdminGroupMember{}
	}
	return group, err
}
func (m mongoUsers) AuditEntries(ctx context.Context, user string) ([]AdminAuditEntry, error) {
	query := bson.M{}
	if user != "" {
		query["$or"] = bson.A{bson.M{"actor_id": user}, bson.M{"target_id": user}}
	}
	// _id is an ObjectID in Go-written audit records; keep its BSON type for paging.
	if page := requestPage(ctx); page != nil && page.Before != nil {
		id, err := bson.ObjectIDFromHex(page.Before.ID)
		if err != nil {
			return nil, ErrModerationConflict
		}
		condition := bson.M{"$or": bson.A{bson.M{"created_at": bson.M{"$lt": page.Before.Time}}, bson.M{"created_at": page.Before.Time, "_id": bson.M{"$lt": id}}}}
		query["$and"] = bson.A{condition}
	}
	cursor, err := m.collection.Database().Collection("admin_logs").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(pageLimit(ctx, 50)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rows := []AdminAuditEntry{}
	if err = cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	rows = trimPage(ctx, rows, func(entry AdminAuditEntry) (time.Time, string) { return entry.Created, entry.ID.Hex() })
	return rows, nil
}
func (m mongoUsers) AdminSummary(ctx context.Context) (map[string]any, error) {
	db := m.collection.Database()
	result := map[string]any{}
	queries := []struct {
		collection, section, key string
		filter                   bson.M
	}{
		{"users", "users", "total", bson.M{}},
		{"users", "users", "active", bson.M{"status": "active"}},
		{"ride_requests", "rides", "pending_requests", bson.M{"status": bson.M{"$in": []string{"pending", "matching"}}, "expires_at": bson.M{"$gt": time.Now().UTC()}}},
		{"ride_groups", "rides", "active_groups", bson.M{"status": bson.M{"$in": []string{"pending", "confirming", "active", "in_progress"}}}},
		{"reports", "reports", "pending", bson.M{"status": "pending"}},
	}
	for _, query := range queries {
		count, err := db.Collection(query.collection).CountDocuments(ctx, query.filter)
		if err != nil {
			return nil, err
		}
		section, ok := result[query.section].(map[string]int64)
		if !ok {
			section = map[string]int64{}
			result[query.section] = section
		}
		section[query.key] = count
	}
	state, err := m.Maintenance(ctx)
	if err != nil {
		return nil, err
	}
	result["maintenance_mode"] = state.Enabled
	return result, nil
}
func (a *AuthDependencies) adminRead(action string) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		repository, ok := a.Users.(AdminReadRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Admin storage unavailable"})
			return
		}
		var result any
		var err error
		switch action {
		case "lookup":
			email, id := strings.TrimSpace(r.URL.Query().Get("email")), strings.TrimSpace(r.URL.Query().Get("user_id"))
			if (email == "") == (id == "") || len(email) > 320 || len(id) > 100 {
				writeJSON(w, 422, map[string]string{"detail": "Provide exactly one email or user_id"})
				return
			}
			field, value := "user_id", id
			if email != "" {
				field, value = "email", strings.ToLower(email)
			}
			var account User
			account, err = repository.LookupAccount(r.Context(), field, value)
			result = profile(account)
		case "group":
			id := r.PathValue("group_id")
			if id == "" || len(id) > 100 {
				writeJSON(w, 422, map[string]string{"detail": "Invalid group ID"})
				return
			}
			result, err = repository.AdminGroup(r.Context(), id)
		case "logs":
			id := r.PathValue("user_id")
			if len(id) > 100 {
				writeJSON(w, 422, map[string]string{"detail": "Invalid user ID"})
				return
			}
			paged, valid := preparePage(w, r, "admin-audit:"+id, 50)
			if !valid {
				return
			}
			r = paged
			result, err = repository.AuditEntries(r.Context(), id)
		case "stats":
			result, err = repository.AdminSummary(r.Context())
		}
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			writeJSON(w, 404, map[string]string{"detail": "Record not found"})
		case errors.Is(err, ErrModerationConflict):
			writeJSON(w, 409, map[string]string{"detail": "Use a unique user ID or a valid audit cursor"})
		case err != nil:
			writeJSON(w, 503, map[string]string{"detail": "Unable to load admin records"})
		default:
			writePageHeader(w, r)
			writeJSON(w, 200, result)
		}
	}
}
