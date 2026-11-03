package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type BlockRecord struct {
	Blocker string    `json:"-" bson:"blocker_user_id"`
	Target  string    `json:"user_id" bson:"blocked_user_id"`
	Name    string    `json:"display_name,omitempty" bson:"-"`
	Email   string    `json:"email" bson:"-"`
	Created time.Time `json:"created_at" bson:"created_at"`
}
type SafetyRepository interface {
	BlockedPeers(context.Context, string) (map[string]bool, error)
	MyBlocks(context.Context, string) ([]BlockRecord, error)
	ChangeBlock(context.Context, string, string, bool) error
}

func (m mongoUsers) BlockedPeers(ctx context.Context, user string) (map[string]bool, error) {
	cursor, err := m.collection.Database().Collection("user_blocks").Find(ctx, bson.M{"$or": bson.A{bson.M{"blocker_user_id": user}, bson.M{"blocked_user_id": user}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	peers := map[string]bool{}
	for cursor.Next(ctx) {
		var block BlockRecord
		if err = cursor.Decode(&block); err != nil {
			return nil, err
		}
		peer := block.Target
		if peer == user {
			peer = block.Blocker
		}
		peers[peer] = true
	}
	return peers, cursor.Err()
}
func (m mongoUsers) MyBlocks(ctx context.Context, user string) ([]BlockRecord, error) {
	cursor, err := m.collection.Database().Collection("user_blocks").Find(ctx, bson.M{"blocker_user_id": user}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(1000))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	records := []BlockRecord{}
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.Target)
	}
	if len(ids) == 0 {
		return records, nil
	}
	profiles, err := m.collection.Find(ctx, bson.M{"user_id": bson.M{"$in": ids}}, options.Find().SetProjection(bson.M{"user_id": 1, "display_name": 1, "email": 1}))
	if err != nil {
		return nil, err
	}
	defer profiles.Close(ctx)
	names := map[string]string{}
	emails := map[string]string{}
	for profiles.Next(ctx) {
		var profile struct {
			ID    string `bson:"user_id"`
			Name  string `bson:"display_name"`
			Email string `bson:"email"`
		}
		if err := profiles.Decode(&profile); err != nil {
			return nil, err
		}
		names[profile.ID] = profile.Name
		emails[profile.ID] = profile.Email
	}
	if err := profiles.Err(); err != nil {
		return nil, err
	}
	for i := range records {
		records[i].Name = names[records[i].Target]
		records[i].Email = emails[records[i].Target]
	}
	return records, nil
}
func (m mongoUsers) ChangeBlock(ctx context.Context, user, target string, blocked bool) error {
	if user == target || target == "" {
		return ErrModerationConflict
	}
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		now := time.Now().UTC()
		// Both profile documents are also touched by joining. This prevents a block
		// and a membership reservation from committing on incompatible snapshots.
		for _, id := range []string{user, target} {
			result, err := m.collection.UpdateOne(tx, bson.M{"user_id": id}, bson.M{"$inc": bson.M{"safety_version": 1}})
			if err != nil {
				return nil, err
			}
			if result.MatchedCount != 1 {
				return nil, mongo.ErrNoDocuments
			}
		}
		filter := bson.M{"blocker_user_id": user, "blocked_user_id": target}
		if !blocked {
			_, err := db.Collection("user_blocks").DeleteOne(tx, filter)
			return nil, err
		}
		if _, err := db.Collection("user_blocks").UpdateOne(tx, filter, bson.M{"$setOnInsert": bson.M{"created_at": now}}, options.UpdateOne().SetUpsert(true)); err != nil {
			return nil, err
		}
		cursor, err := db.Collection("ride_groups").Find(tx, bson.M{"members.user_id": bson.M{"$all": []string{user, target}}, "status": bson.M{"$in": []string{"pending", "confirming", "active", "in_progress"}}})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(tx)
		var groups []GroupRecord
		if err = cursor.All(tx, &groups); err != nil {
			return nil, err
		}
		for _, group := range groups {
			if _, err = db.Collection("ride_groups").UpdateOne(tx, bson.M{"group_id": group.ID}, bson.M{"$set": bson.M{"status": "cancelled", "updated_at": now}}); err != nil {
				return nil, err
			}
			if _, err = m.rides().UpdateMany(tx, bson.M{"group_id": group.ID, "status": "matched"}, bson.M{"$set": bson.M{"status": "cancelled"}}); err != nil {
				return nil, err
			}
			if _, err = db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "group_cancelled", "group_id": group.ID, "status": "pending", "attempts": 0, "created_at": now}); err != nil {
				return nil, err
			}
		}
		_, err = WriteAudit(tx, db, bson.M{"actor_id": user, "action": "blocked_user", "target_id": target, "created_at": now})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}
func (a *AuthDependencies) blocks(action string) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		repository, ok := a.Users.(SafetyRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Safety settings unavailable"})
			return
		}
		if action == "list" {
			blocks, err := repository.MyBlocks(r.Context(), user.ID)
			if err != nil {
				writeJSON(w, 503, map[string]string{"detail": "Unable to load blocked users"})
				return
			}
			writeJSON(w, 200, map[string]any{"blocked_users": blocks})
			return
		}
		target := r.PathValue("user_id")
		if target == user.ID || target == "" || len(target) > 100 {
			writeJSON(w, 422, map[string]string{"detail": "Choose another user"})
			return
		}
		err := repository.ChangeBlock(r.Context(), user.ID, target, action == "block")
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeJSON(w, 404, map[string]string{"detail": "User not found"})
			return
		}
		if err != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to update block settings"})
			return
		}
		writeJSON(w, 200, map[string]bool{"success": true, "blocked": action == "block"})
	}
}
