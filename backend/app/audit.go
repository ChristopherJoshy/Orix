package server

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// WriteAudit is shared by API transactions and privileged CLI commands. Legacy
// databases have a unique log_id index: omitting that field would allow only one
// new audit before subsequent actions fail with a duplicate-null key. Keep both
// timestamp names while historical readers/data migrate to created_at.
func WriteAudit(ctx context.Context, db *mongo.Database, fields bson.M) (*mongo.InsertOneResult, error) {
	entry := make(bson.M, len(fields)+4)
	for key, value := range fields {
		entry[key] = value
	}
	id := bson.NewObjectID()
	entry["_id"] = id
	entry["log_id"] = id.Hex()
	created, ok := entry["created_at"].(time.Time)
	if !ok || created.IsZero() {
		created = time.Now().UTC()
	}
	entry["created_at"] = created.UTC()
	entry["timestamp"] = created.UTC()
	return db.Collection("admin_logs").InsertOne(ctx, entry)
}
