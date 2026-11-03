package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// removeTestAccounts only touches the isolated, loopback internal-test database.
func removeTestAccounts() error {
	if err := validateLocalEmulator(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		return err
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = client.Disconnect(cleanup)
	}()
	db := client.Database("orix_internal_test")
	firebaseIDs := []string{}
	for _, persona := range testPersonas {
		firebaseIDs = append(firebaseIDs, "orix-test-"+persona.slug)
	}
	cursor, err := db.Collection("users").Find(ctx, bson.M{"firebase_uid": bson.M{"$in": firebaseIDs}})
	if err != nil {
		return err
	}
	var users []struct {
		ID string `bson:"user_id"`
	}
	if err = cursor.All(ctx, &users); err != nil {
		return err
	}
	ids := []string{}
	known := map[string]bool{}
	for _, user := range users {
		ids = append(ids, user.ID)
		known[user.ID] = true
	}
	groupIDs := []string{}
	if len(ids) > 0 {
		groupsCursor, err := db.Collection("ride_groups").Find(ctx, bson.M{"$or": bson.A{bson.M{"members.user_id": bson.M{"$in": ids}}, bson.M{"origin_user_id": bson.M{"$in": ids}}}})
		if err != nil {
			return err
		}
		var groups []struct {
			ID      string `bson:"group_id"`
			Members []struct {
				ID string `bson:"user_id"`
			} `bson:"members"`
		}
		if err = groupsCursor.All(ctx, &groups); err != nil {
			return err
		}
		for _, group := range groups {
			for _, member := range group.Members {
				if !known[member.ID] {
					return errors.New("test group contains a non-seeded user; leave data intact for manual review")
				}
			}
			groupIDs = append(groupIDs, group.ID)
		}
		// Operations are idempotent. Delete users last so interrupted cleanup can retry.
		groupFilter := bson.M{"group_id": bson.M{"$in": groupIDs}}
		for _, name := range []string{"chat_messages"} {
			if _, err = db.Collection(name).DeleteMany(ctx, groupFilter); err != nil {
				return err
			}
		}
		for _, name := range []string{"ride_requests", "notifications", "user_logs", "support_tickets", "user_avatars"} {
			if _, err = db.Collection(name).DeleteMany(ctx, bson.M{"user_id": bson.M{"$in": ids}}); err != nil {
				return err
			}
		}
		// Outbox actors and direct targets use different fields; group events may
		// have neither. Remove all three scopes before identities disappear.
		if _, err = db.Collection("notification_outbox").DeleteMany(ctx, bson.M{"$or": bson.A{
			bson.M{"user_id": bson.M{"$in": ids}},
			bson.M{"target_user_id": bson.M{"$in": ids}},
			bson.M{"group_id": bson.M{"$in": groupIDs}},
		}}); err != nil {
			return err
		}
		if _, err = db.Collection("account_deletions").DeleteMany(ctx, bson.M{"$or": bson.A{
			bson.M{"_id": bson.M{"$in": ids}},
			bson.M{"firebase_uid": bson.M{"$in": firebaseIDs}},
		}}); err != nil {
			return err
		}
		_, err = db.Collection("reports").DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{"reporter_user_id": bson.M{"$in": ids}}, bson.M{"accused_user_id": bson.M{"$in": ids}}}})
		if err != nil {
			return err
		}
		_, err = db.Collection("ratings").DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{"rater_user_id": bson.M{"$in": ids}}, bson.M{"rated_user_id": bson.M{"$in": ids}}}})
		if err != nil {
			return err
		}
		_, err = db.Collection("admin_logs").DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{"actor_id": bson.M{"$in": ids}}, bson.M{"target_id": bson.M{"$in": ids}}}})
		if err != nil {
			return err
		}
		_, err = db.Collection("user_blocks").DeleteMany(ctx, bson.M{"$or": bson.A{bson.M{"blocker_user_id": bson.M{"$in": ids}}, bson.M{"blocked_user_id": bson.M{"$in": ids}}}})
		if err != nil {
			return err
		}
		// Preserve group membership until dependent cleanup succeeds, so a retry
		// can rediscover the same group-scoped outbox/chat records.
		if _, err = db.Collection("ride_groups").DeleteMany(ctx, groupFilter); err != nil {
			return err
		}
		_, err = db.Collection("users").DeleteMany(ctx, bson.M{"firebase_uid": bson.M{"$in": firebaseIDs}})
		if err != nil {
			return err
		}
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: "demo-orix"})
	if err != nil {
		return err
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		return err
	}
	for _, id := range firebaseIDs {
		if err = authClient.DeleteUser(ctx, id); err != nil && !auth.IsUserNotFound(err) {
			return errors.New("emulator account deletion failed; rerun cleanup")
		}
	}
	fmt.Printf("Removed %d seeded emulator accounts and their local app records.\n", len(testPersonas))
	return nil
}
