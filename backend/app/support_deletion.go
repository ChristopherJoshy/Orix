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

type SupportDeletionRequest struct {
	ActorFirebaseUID  string
	TargetFirebaseUID string
	CaseID            string
}
type SupportDeletionPlan struct {
	ActorID       string `json:"actor_id"`
	TargetID      string `json:"target_id"`
	HasAppAccount bool   `json:"has_app_account"`
	CaseID        string `json:"case_id"`
	Applied       bool   `json:"applied"`
}

var supportDeletionCase = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,99}$`)

func (input SupportDeletionRequest) Validate() error {
	if input.ActorFirebaseUID == "" || input.TargetFirebaseUID == "" || len(input.ActorFirebaseUID) > 128 || len(input.TargetFirebaseUID) > 128 || input.ActorFirebaseUID == input.TargetFirebaseUID || !supportDeletionCase.MatchString(input.CaseID) {
		return errors.New("support deletion requires distinct Firebase UIDs and an opaque case reference")
	}
	return nil
}

// ReviewSupportDeletion is an operator-only entry point, not an HTTP route.
// Its CLI caller validates Firebase operator status and explicit environment /
// ownership-review assertions before apply. The database role is rechecked here.
// Planning does not upsert profiles, create indexes, enqueue jobs or write audits.
func ReviewSupportDeletion(ctx context.Context, db *mongo.Database, input SupportDeletionRequest, apply bool) (SupportDeletionPlan, error) {
	if err := input.Validate(); err != nil {
		return SupportDeletionPlan{}, err
	}

	m := mongoUsers{collection: db.Collection("users")}
	lookup := func(tx context.Context) (SupportDeletionPlan, error) {
		var actor User
		if err := m.collection.FindOne(tx, bson.M{"firebase_uid": input.ActorFirebaseUID, "role": "admin", "status": "active"}).Decode(&actor); err != nil {
			return SupportDeletionPlan{}, err
		}
		if accountRestricted(actor) {
			return SupportDeletionPlan{}, ErrModerationConflict
		}
		plan := SupportDeletionPlan{ActorID: actor.ID, CaseID: input.CaseID}
		var target User
		err := m.collection.FindOne(tx, bson.M{"firebase_uid": input.TargetFirebaseUID}).Decode(&target)
		if errors.Is(err, mongo.ErrNoDocuments) {
			// A Firebase identity may exist without an app profile. Queue only its
			// deletion; never create an app account just to remove the provider identity.
			plan.TargetID = stableID("firebase-only:" + input.TargetFirebaseUID)
		} else if err != nil {
			return plan, err
		} else {
			plan.TargetID, plan.HasAppAccount = target.ID, true
		}
		return plan, nil
	}
	if !apply {
		return lookup(ctx)
	}
	session, err := db.Client().StartSession()
	if err != nil {
		return SupportDeletionPlan{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		plan, err := lookup(tx)
		if err != nil {
			return nil, err
		}
		if _, err := m.authorizeAdminMutation(tx, plan.ActorID); err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		if plan.HasAppAccount {
			if err := m.queueAccountDeletion(tx, plan.TargetID); err != nil {
				return nil, err
			}
		} else {
			if _, err := db.Collection("account_deletions").UpdateOne(tx, bson.M{"_id": plan.TargetID}, bson.M{"$setOnInsert": bson.M{
				"firebase_uid": input.TargetFirebaseUID, "status": "pending", "created_at": now, "attempts": 0,
			}}, options.UpdateOne().SetUpsert(true)); err != nil {
				return nil, err
			}
		}
		var job struct {
			Status string `bson:"status"`
			CaseID string `bson:"support_case_id"`
		}
		if err := db.Collection("account_deletions").FindOne(tx, bson.M{"_id": plan.TargetID}).Decode(&job); err != nil {
			return nil, err
		}
		if job.CaseID != "" && job.CaseID != input.CaseID {
			return nil, errors.New("a different support case already owns this deletion request")
		}
		if job.Status != "completed" && job.CaseID == "" {
			if _, err := db.Collection("account_deletions").UpdateOne(tx, bson.M{"_id": plan.TargetID}, bson.M{"$set": bson.M{
				"support_case_id": input.CaseID, "reviewed_by": plan.ActorID, "reviewed_at": now,
			}}); err != nil {
				return nil, err
			}
			if _, err := WriteAudit(tx, db, bson.M{"actor_id": plan.ActorID, "target_id": plan.TargetID, "action": "support_account_deletion_requested", "case_id": input.CaseID, "created_at": now}); err != nil {
				return nil, err
			}
		}
		plan.Applied = true
		return plan, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return SupportDeletionPlan{}, err
	}
	return result.(SupportDeletionPlan), nil
}
