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

type GroupRejectionRepository interface {
	RejectGroup(context.Context, string, string) (string, error)
}

// Requeue the existing requests atomically. No new ride or duplicate assignment
// is created. Exclusions belong to this journey, not the permanent block list.
func (m mongoUsers) RejectGroup(ctx context.Context, actor, id string) (string, error) {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return "", err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		now := time.Now().UTC()
		var actorProfile User
		if err := m.collection.FindOne(tx, bson.M{"user_id": actor}).Decode(&actorProfile); err != nil {
			return nil, err
		}
		if !actorProfile.Onboarded || accountRestricted(actorProfile) {
			return nil, ErrGroupUnavailable
		}
		state, err := m.Maintenance(tx)
		if err != nil {
			return nil, err
		}
		if state.Enabled {
			return nil, ErrGroupUnavailable
		}
		var stored struct {
			GroupRecord `bson:",inline"`
			RejectedBy  string `bson:"rejected_by"`
			RequestID   string `bson:"rejection_request_id"`
		}
		if err := db.Collection("ride_groups").FindOne(tx, bson.M{"group_id": id, "members.user_id": actor}).Decode(&stored); err != nil {
			return nil, err
		}
		if stored.RejectedBy == actor && stored.RequestID != "" {
			return stored.RequestID, nil
		}
		group := stored.GroupRecord
		if !canRejectGroup(group, now) {
			return nil, ErrGroupUnavailable
		}
		requestID := ""
		for _, member := range group.Members {
			// Same documents are touched by assignment, account safety and rejection.
			locked, err := m.collection.UpdateOne(tx, bson.M{"user_id": member.UserID}, bson.M{"$inc": bson.M{"safety_version": 1}})
			if err != nil {
				return nil, err
			}
			if locked.MatchedCount != 1 {
				return nil, ErrGroupUnavailable
			}
			var ride Ride
			if err := m.rides().FindOne(tx, bson.M{"group_id": id, "user_id": member.UserID, "status": "matched"}).Decode(&ride); err != nil {
				return nil, err
			}
			end := ride.EndUTC
			if end.IsZero() {
				end = group.End
			}
			if !end.After(now) {
				return nil, ErrGroupUnavailable
			}
			changes := bson.M{"status": "pending", "group_id": nil, "expires_at": minTime(now.Add(15*time.Hour), end), "next_match_at": now, "updated_at": now}
			update := bson.M{"$set": changes, "$unset": bson.M{"match_lease": "", "match_lease_until": "", "match_error": "", "candidate_scan_after": ""}}
			if member.UserID == actor {
				if len(ride.RejectedGroups) >= 100 {
					return nil, ErrTicketLimit
				}
				peers := make([]string, 0, len(group.Members)-1)
				for _, other := range group.Members {
					if other.UserID != actor {
						peers = append(peers, other.UserID)
					}
				}
				update["$addToSet"] = bson.M{"rejected_group_ids": id, "rejected_user_ids": bson.M{"$each": peers}}
				requestID = ride.ID
			}
			changed, err := m.rides().UpdateOne(tx, bson.M{"request_id": ride.ID, "status": "matched", "group_id": id}, update)
			if err != nil {
				return nil, err
			}
			if changed.MatchedCount != 1 {
				return nil, ErrGroupUnavailable
			}
		}
		if requestID == "" {
			return nil, ErrGroupUnavailable
		}
		changed, err := db.Collection("ride_groups").UpdateOne(tx, bson.M{"group_id": id, "status": group.Status}, bson.M{"$set": bson.M{
			"status": "cancelled", "rejected_by": actor, "rejection_request_id": requestID, "updated_at": now, "reminder_schedule": nil, "road_plan": nil,
		}, "$unset": bson.M{"match_lease": "", "match_lease_until": "", "next_match_at": ""}})
		if err != nil {
			return nil, err
		}
		if changed.MatchedCount != 1 {
			return nil, ErrGroupUnavailable
		}
		if _, err = db.Collection("notification_outbox").InsertOne(tx, bson.M{"_id": stableID("group-rejected:" + id), "type": "group_rejected", "group_id": id, "status": "pending", "attempts": 0, "created_at": now}); err != nil {
			return nil, err
		}
		_, err = WriteAudit(tx, db, bson.M{"actor_id": actor, "action": "rejected_group", "target_id": id, "created_at": now})
		return requestID, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return "", err
	}
	return result.(string), nil
}
func canRejectGroup(group GroupRecord, now time.Time) bool {
	if len(group.Members) < 2 || !group.End.After(now) {
		return false
	}
	return group.Status == "pending" || group.Status == "confirming" || group.Status == "active"
}
func (a *AuthDependencies) rejectGroup(w http.ResponseWriter, r *http.Request, user User) {
	if !user.Onboarded {
		writeJSON(w, 403, map[string]string{"detail": "Complete onboarding first"})
		return
	}
	repo, ok := a.Users.(GroupRejectionRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Matching restart unavailable"})
		return
	}
	id, err := repo.RejectGroup(r.Context(), user.ID, r.PathValue("group_id"))
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Group not found"})
		return
	}
	if errors.Is(err, ErrGroupUnavailable) {
		writeJSON(w, 409, map[string]string{"detail": "This group cannot be rejected now. Refresh its current status."})
		return
	}
	if errors.Is(err, ErrTicketLimit) {
		writeJSON(w, 409, map[string]string{"detail": "This journey reached its rejection limit. Cancel it before planning a new journey."})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Matching restart was not confirmed. Refresh before retrying."})
		return
	}
	writeJSON(w, 200, map[string]any{"success": true, "request_id": id, "matching_restarted": true})
}
