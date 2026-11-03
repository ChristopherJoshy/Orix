package server

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type LifecycleWorker interface{ RunLifecycle(context.Context) }

func expiryReason(group GroupRecord, now time.Time) string {
	switch group.Status {
	case "pending":
		searchEnd := group.SearchExpires
		if searchEnd.IsZero() {
			searchEnd = group.Created.Add(15 * time.Hour)
		}
		if !searchEnd.IsZero() && !searchEnd.After(now) {
			return "search_expired"
		}
	case "confirming":
		if !group.Deadline.IsZero() && !group.Deadline.After(now) {
			return "confirmation_expired"
		}
	case "active":
	default:
		return ""
	}
	if !group.End.IsZero() && !group.End.After(now) {
		return "departure_window_expired"
	}
	return ""
}
func (m mongoUsers) recordWorker(ctx context.Context, name string, workErr error) {
	if ctx.Err() != nil {
		return
	}
	report, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	fields := bson.M{"heartbeat_at": time.Now().UTC()}
	if workErr != nil {
		fields["last_error"] = "Background work failed; retry scheduled"
		fields["last_error_at"] = time.Now().UTC()
	} else {
		fields["last_success_at"] = time.Now().UTC()
		fields["last_error"] = ""
	}
	_, _ = m.collection.Database().Collection("worker_health").UpdateOne(report, bson.M{"_id": name}, bson.M{"$set": fields}, options.UpdateOne().SetUpsert(true))
}
func (m mongoUsers) RunLifecycle(ctx context.Context) {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			work, cancel := context.WithTimeout(ctx, 25*time.Second)
			err := m.expireCoordination(work)
			cancel()
			m.recordWorker(ctx, "lifecycle", err)
		}
	}
}
func (m mongoUsers) expireCoordination(ctx context.Context) error {
	now := time.Now().UTC()
	db := m.collection.Database()
	if _, err := m.rides().UpdateMany(ctx, bson.M{"status": bson.M{"$in": []string{"pending", "matching"}}, "expires_at": bson.M{"$lte": now}}, bson.M{"$set": bson.M{"status": "expired", "expired_at": now}}); err != nil {
		return err
	}
	due := bson.A{
		bson.M{"status": "pending", "search_expires_at": bson.M{"$lte": now}},
		bson.M{"status": "pending", "search_expires_at": bson.M{"$exists": false}, "created_at": bson.M{"$lte": now.Add(-15 * time.Hour)}},
		bson.M{"status": "confirming", "confirmation_deadline": bson.M{"$lte": now}},
		bson.M{"status": bson.M{"$in": []string{"pending", "confirming", "active"}}, "departure_end": bson.M{"$lte": now}},
	}
	cursor, err := db.Collection("ride_groups").Find(ctx, bson.M{"$or": due}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(50))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	var groups []GroupRecord
	if err = cursor.All(ctx, &groups); err != nil {
		return err
	}
	for _, group := range groups {
		if err = m.expireGroup(ctx, group.ID); err != nil {
			return err
		}
	}
	return nil
}
func (m mongoUsers) expireGroup(ctx context.Context, id string) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		var group GroupRecord
		if err := db.Collection("ride_groups").FindOne(tx, bson.M{"group_id": id}).Decode(&group); errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		reason := expiryReason(group, now)
		if reason == "" {
			return nil, nil
		}
		if _, err := db.Collection("ride_groups").UpdateOne(tx, bson.M{"group_id": id}, bson.M{"$set": bson.M{"status": "expired", "expiry_reason": reason, "expired_at": now, "updated_at": now, "road_plan": nil, "reminder_schedule": nil}}); err != nil {
			return nil, err
		}
		if _, err := m.rides().UpdateMany(tx, bson.M{"group_id": id, "status": "matched"}, bson.M{"$set": bson.M{"status": "expired", "expired_at": now}}); err != nil {
			return nil, err
		}
		_, err := db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "group_cancelled", "group_id": id, "status": "pending", "attempts": 0, "created_at": now})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}
