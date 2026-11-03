package server

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type LegacyCollectionPlan struct {
	Name    string `json:"name"`
	Records int64  `json:"records"`
	Removed int64  `json:"removed"`
}
type LegacyRetirementPlan struct {
	ActorID         string                 `json:"actor_id"`
	TestReference   string                 `json:"test_reference,omitempty"`
	Collections     []LegacyCollectionPlan `json:"collections"`
	LegacyProfiles  int64                  `json:"legacy_profiles"`
	ProfilesCleaned int64                  `json:"profiles_cleaned"`
	Applied         bool                   `json:"applied"`
}

var retirementReference = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,99}$`)

// These are the exact retired bot/domain collections found in the legacy source.
// Active app data and app_config/version are deliberately outside this allowlist.
func retiredCollections() []string {
	return []string{"ai_chat_context", "ai_pending_confirmations", "ai_config", "promo_config", "telegram_admins", "telegram_head_admins", "domain_names", "domain_requests"}
}
func legacyRiderFields() bson.M {
	return bson.M{"$or": bson.A{bson.M{"is_rider": bson.M{"$exists": true}}, bson.M{"rider_info": bson.M{"$exists": true}}}}
}

// RetireLegacyData is called only by the privileged CLI. The caller checks
// Firebase operator identity plus explicit database/stopped-writer/test assertions.
// Planning performs reads only. Apply is incremental with a fresh authorization
// check and audit in every bounded batch; an interrupted run can safely retry.
func RetireLegacyData(ctx context.Context, db *mongo.Database, actorUID, testReference string, apply bool) (LegacyRetirementPlan, error) {
	plan := LegacyRetirementPlan{TestReference: testReference, Collections: []LegacyCollectionPlan{}}
	if actorUID == "" || len(actorUID) > 128 || (testReference != "" && !retirementReference.MatchString(testReference)) || (apply && testReference == "") {
		return plan, errors.New("retirement requires an administrator UID and an opaque test reference for apply")
	}
	m := mongoUsers{collection: db.Collection("users")}
	var actor User
	if err := m.collection.FindOne(ctx, bson.M{"firebase_uid": actorUID, "role": "admin", "status": "active"}).Decode(&actor); err != nil {
		return plan, err
	}
	if accountRestricted(actor) {
		return plan, ErrAdminAccess
	}
	plan.ActorID = actor.ID
	for _, name := range retiredCollections() {
		count, err := db.Collection(name).CountDocuments(ctx, bson.M{})
		if err != nil {
			return plan, err
		}
		plan.Collections = append(plan.Collections, LegacyCollectionPlan{Name: name, Records: count})
	}
	var err error
	plan.LegacyProfiles, err = m.collection.CountDocuments(ctx, legacyRiderFields())
	if err != nil || !apply {
		return plan, err
	}
	for index := range plan.Collections {
		for {
			count, err := m.retireLegacyBatch(ctx, plan.ActorID, plan.Collections[index].Name, testReference, false)
			plan.Collections[index].Removed += count
			if err != nil {
				return plan, err
			}
			if count == 0 {
				break
			}
		}
	}
	for {
		count, err := m.retireLegacyBatch(ctx, plan.ActorID, "users", testReference, true)
		plan.ProfilesCleaned += count
		if err != nil {
			return plan, err
		}
		if count == 0 {
			break
		}
	}
	plan.Applied = true
	return plan, nil
}
func (m mongoUsers) retireLegacyBatch(ctx context.Context, actor, name, reference string, profiles bool) (int64, error) {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return 0, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		collection := m.collection.Database().Collection(name)
		filter := bson.M{}
		if profiles {
			filter = legacyRiderFields()
		}
		cursor, err := collection.Find(tx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}).SetLimit(500).SetProjection(bson.M{"_id": 1}))
		if err != nil {
			return nil, err
		}
		var rows []struct {
			ID any `bson:"_id"`
		}
		if err = cursor.All(tx, &rows); err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return int64(0), nil
		}
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		ids := make([]any, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		var count int64
		selected := bson.M{"_id": bson.M{"$in": ids}}
		if profiles {
			changed, err := collection.UpdateMany(tx, selected, bson.M{"$unset": bson.M{"is_rider": "", "rider_info": ""}})
			if err != nil {
				return nil, err
			}
			count = changed.ModifiedCount
		} else {
			removed, err := collection.DeleteMany(tx, selected)
			if err != nil {
				return nil, err
			}
			count = removed.DeletedCount
		}
		if count != int64(len(rows)) {
			return nil, errors.New("legacy retirement encountered concurrent changes; stop writers and re-plan")
		}
		action := "retired_legacy_records"
		if profiles {
			action = "retired_legacy_rider_fields"
		}
		if _, err := WriteAudit(tx, m.collection.Database(), bson.M{"actor_id": actor, "target_id": name, "action": action, "count": count, "test_reference": reference, "created_at": time.Now().UTC()}); err != nil {
			return nil, err
		}
		return count, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}
