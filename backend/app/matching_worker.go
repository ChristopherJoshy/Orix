package server

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MatchingWorker interface{ RunMatchmaking(context.Context) }

func (m mongoUsers) RunMatchmaking(ctx context.Context) {
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			work, cancel := context.WithTimeout(ctx, 30*time.Second)
			requestErr := m.matchWaitingRequests(work)
			var groupErr error
			if work.Err() == nil {
				groupErr = m.mergeWaitingGroups(work)
			}
			err := errors.Join(requestErr, groupErr)
			cancel()
			m.recordWorker(ctx, "matching", err)
		}
	}
}
func dueMatching(now time.Time) bson.A {
	return bson.A{
		bson.M{"$or": bson.A{bson.M{"next_match_at": bson.M{"$lte": now}}, bson.M{"next_match_at": bson.M{"$exists": false}}}},
		bson.M{"$or": bson.A{bson.M{"match_lease_until": bson.M{"$lte": now}}, bson.M{"match_lease_until": bson.M{"$exists": false}}}},
	}
}
func (m mongoUsers) matchWaitingRequests(ctx context.Context) error {
	state, err := m.Maintenance(ctx)
	if err != nil {
		return err
	}
	if state.Enabled {
		return nil
	}
	var firstErr error
	for range 10 {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		now := time.Now().UTC()
		lease := stableID(now.String())
		var ride Ride
		query := bson.M{"status": bson.M{"$in": []string{"pending", "matching"}}, "expires_at": bson.M{"$gt": now}, "$and": dueMatching(now)}
		err := m.rides().FindOneAndUpdate(ctx, query, bson.M{"$set": bson.M{"status": "matching", "match_lease": lease, "match_lease_until": now.Add(60 * time.Second), "next_match_at": now.Add(15 * time.Second)}}, options.FindOneAndUpdate().SetSort(bson.D{{Key: "next_match_at", Value: 1}, {Key: "created_at", Value: 1}, {Key: "request_id", Value: 1}}).SetReturnDocument(options.After)).Decode(&ride)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return firstErr
		}
		if err != nil {
			return err
		}
		matchErr := m.tryMatchRide(ctx, ride, true)
		update := bson.M{"$unset": bson.M{"match_lease": "", "match_lease_until": ""}}
		if matchErr != nil {
			if firstErr == nil {
				firstErr = matchErr
			}
			update["$set"] = bson.M{"next_match_at": time.Now().Add(30 * time.Second), "match_error": "Matching will retry"}
		} else {
			update["$unset"].(bson.M)["match_error"] = ""
		}
		if _, err = m.rides().UpdateOne(ctx, bson.M{"request_id": ride.ID, "match_lease": lease}, update); err != nil {
			return err
		}
	}
	return firstErr
}
func (m mongoUsers) tryMatchRide(ctx context.Context, ride Ride, createSolo bool) error {
	var owner User
	if err := m.collection.FindOne(ctx, bson.M{"user_id": ride.UserID}).Decode(&owner); err != nil {
		return err
	}
	if accountRestricted(owner) || !owner.Onboarded {
		return ErrGroupUnavailable
	}
	scan := &candidateScan{After: ride.CandidateAfter}
	groups, err := m.CandidateGroups(context.WithValue(ctx, candidateScanKey{}, scan), ride)
	if err != nil {
		return err
	}
	// Save bounded progress even when this batch has no compatible groups.
	// Compare the previous position so stale workers cannot rewind a newer scan.
	filter := bson.M{"request_id": ride.ID, "user_id": owner.ID, "status": ride.Status}
	if ride.CandidateAfter == "" {
		filter["$or"] = bson.A{bson.M{"candidate_scan_after": ""}, bson.M{"candidate_scan_after": bson.M{"$exists": false}}}
	} else {
		filter["candidate_scan_after"] = ride.CandidateAfter
	}
	update := bson.M{"$unset": bson.M{"candidate_scan_after": ""}}
	if scan.Next != "" {
		update = bson.M{"$set": bson.M{"candidate_scan_after": scan.Next}}
	}
	if _, err = m.rides().UpdateOne(ctx, filter, update); err != nil {
		return err
	}
	ranked, err := rankGroups(ride, owner, groups, time.Now().UTC())
	if err != nil {
		return err
	}
	var candidateErr error
	// Try a bounded number of current candidates; reservation rechecks snapshots.
	for i, candidate := range ranked {
		if i == 8 {
			break
		}
		if _, err = m.AssignGroup(ctx, owner.ID, ride.ID, candidate.GroupID); err == nil {
			return nil
		}
		if !errors.Is(err, ErrGroupUnavailable) && !errors.Is(err, ErrNoRoadPlan) && !errors.Is(err, mongo.ErrNoDocuments) && candidateErr == nil {
			candidateErr = err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	if createSolo {
		_, err = m.AssignGroup(ctx, owner.ID, ride.ID, "")
		return err
	}
	return candidateErr
}
func (m mongoUsers) mergeWaitingGroups(ctx context.Context) error {
	state, err := m.Maintenance(ctx)
	if err != nil {
		return err
	}
	if state.Enabled {
		return nil
	}
	groups := m.collection.Database().Collection("ride_groups")
	var firstErr error
	for range 10 {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		now := time.Now().UTC()
		lease := stableID(now.String())
		var group GroupRecord
		query := bson.M{"status": "pending", "members": bson.M{"$size": 1}, "members.0.readiness_confirmed": false, "departure_end": bson.M{"$gt": now}, "$and": dueMatching(now)}
		err := groups.FindOneAndUpdate(ctx, query, bson.M{"$set": bson.M{"match_lease": lease, "match_lease_until": now.Add(60 * time.Second), "next_match_at": now.Add(15 * time.Second)}}, options.FindOneAndUpdate().SetSort(bson.D{{Key: "next_match_at", Value: 1}, {Key: "created_at", Value: 1}, {Key: "group_id", Value: 1}}).SetReturnDocument(options.After)).Decode(&group)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return firstErr
		}
		if err != nil {
			return err
		}
		var ride Ride
		err = m.rides().FindOne(ctx, bson.M{"user_id": group.Members[0].UserID, "group_id": group.ID, "status": "matched", "expires_at": bson.M{"$gt": now}}).Decode(&ride)
		var matchErr error
		if err == nil {
			matchErr = m.tryMatchRide(ctx, ride, false)
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return err
		}
		update := bson.M{"$unset": bson.M{"match_lease": "", "match_lease_until": ""}}
		if matchErr != nil {
			if firstErr == nil {
				firstErr = matchErr
			}
			update["$set"] = bson.M{"next_match_at": time.Now().Add(30 * time.Second), "match_error": "Matching will retry"}
		} else {
			update["$unset"].(bson.M)["match_error"] = ""
		}
		if _, err = groups.UpdateOne(ctx, bson.M{"group_id": group.ID, "match_lease": lease}, update); err != nil {
			return err
		}
	}
	return firstErr
}
