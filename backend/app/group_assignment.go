package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"orix/backend/internal/matching"
)

var ErrGroupUnavailable = errors.New("group or request is no longer available")
var ErrRoadRoutingRequired = errors.New("lift itinerary requires road verification before joining")

type AssignmentRepository interface {
	AssignGroup(context.Context, string, string, string) (string, error)
}

func (m mongoUsers) AssignGroup(ctx context.Context, userID, requestID, targetGroup string) (string, error) {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return "", err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		now := time.Now().UTC()
		maintenance, err := m.Maintenance(tx)
		if err != nil {
			return nil, err
		}
		if maintenance.Enabled {
			return nil, ErrGroupUnavailable
		}
		ride, err := m.OwnedRide(tx, userID, requestID)
		if err != nil {
			return nil, err
		}
		sourceGroup := ""
		if ride.Status == "matched" && ride.GroupID != nil {
			if targetGroup == "" || targetGroup == *ride.GroupID {
				return *ride.GroupID, nil
			}
			var source GroupRecord
			if err = m.collection.Database().Collection("ride_groups").FindOne(tx, bson.M{"group_id": *ride.GroupID, "status": "pending", "members.user_id": userID}).Decode(&source); err != nil {
				return nil, ErrGroupUnavailable
			}
			if len(source.Members) != 1 || source.Members[0].UserID != userID || source.Members[0].Ready {
				return nil, ErrGroupUnavailable
			}
			sourceGroup = *ride.GroupID
		}
		if (ride.Status != "pending" && ride.Status != "matching" && sourceGroup == "") || !ride.Expires.After(now) {
			return nil, ErrGroupUnavailable
		}
		var owner User
		if err = m.collection.FindOne(tx, bson.M{"user_id": userID}).Decode(&owner); err != nil {
			return nil, err
		}
		if !owner.Onboarded || accountRestricted(owner) {
			return nil, ErrGroupUnavailable
		}
		if _, err = m.collection.UpdateOne(tx, bson.M{"user_id": userID}, bson.M{"$inc": bson.M{"safety_version": 1}}); err != nil {
			return nil, err
		}
		blocked, err := m.BlockedPeers(tx, userID)
		if err != nil {
			return nil, err
		}
		request, err := routeRequest(ride, owner)
		if err != nil {
			return nil, err
		}
		if !request.End.After(now) {
			return nil, ErrGroupUnavailable
		}
		if request.Start.Before(now) {
			request.Start = now
		}
		groupID := targetGroup
		groups := m.collection.Database().Collection("ride_groups")
		deadline := minTime(now.Add(time.Hour), request.End)
		members := []bson.M{}
		departure, end := request.Start, request.End
		femaleOnly := request.FemaleOnly
		var roadPlan *RoadPlan
		assignedMode := matching.AssignedMode(request, nil)
		if targetGroup == "" {
			raw := make([]byte, 16)
			if _, err = rand.Read(raw); err != nil {
				return nil, err
			}
			groupID = hex.EncodeToString(raw)
		} else {
			var group CandidateGroup
			if err = groups.FindOne(tx, bson.M{"group_id": targetGroup, "status": bson.M{"$in": []string{"pending", "confirming"}}}).Decode(&group); err != nil {
				return nil, ErrGroupUnavailable
			}
			if len(group.Members) == 0 || len(group.Members) >= matching.DefaultPolicy().MaxMembers {
				return nil, ErrGroupUnavailable
			}
			// A transaction rechecks every persisted member's current profile and route.
			for _, member := range group.Members {
				if member.UserID == userID || blocked[member.UserID] {
					return nil, ErrGroupUnavailable
				}
				var profile User
				if _, err = m.collection.UpdateOne(tx, bson.M{"user_id": member.UserID}, bson.M{"$inc": bson.M{"safety_version": 1}}); err != nil {
					return nil, err
				}
				var memberRide Ride
				if err = m.collection.FindOne(tx, bson.M{"user_id": member.UserID}).Decode(&profile); err != nil {
					return nil, err
				}
				if !profile.Onboarded || accountRestricted(profile) {
					return nil, ErrGroupUnavailable
				}
				if err = m.rides().FindOne(tx, bson.M{"user_id": member.UserID, "group_id": targetGroup, "status": "matched"}).Decode(&memberRide); err != nil {
					return nil, ErrGroupUnavailable
				}
				route, routeErr := routeRequest(memberRide, profile)
				if routeErr != nil || !route.End.After(now) {
					return nil, ErrGroupUnavailable
				}
				if route.Start.Before(now) {
					route.Start = now
				}
				group.Requests = append(group.Requests, route)
				role := "passenger"
				if route.Driver {
					role = "rider"
				}
				members = append(members, bson.M{"user_id": member.UserID, "role": role, "readiness_confirmed": false, "joined_at": now})
				if route.Start.After(departure) {
					departure = route.Start
				}
				end = minTime(end, route.End)
				femaleOnly = femaleOnly || route.FemaleOnly
			}
			ranked, rankErr := matching.Rank(request, []matching.Group{{ID: group.ID, Members: group.Requests}}, matching.DefaultPolicy())
			if rankErr != nil || len(ranked) == 0 {
				return nil, ErrGroupUnavailable
			}
			assignedMode = matching.AssignedMode(request, group.Requests)
			if matching.RequiresLift(request, group.Requests) {
				roadPlan, err = m.router.LiftPlan(tx, append(group.Requests, request))
				if err != nil {
					return nil, err
				}
			}
		}
		if targetGroup == "" && request.Mode == matching.Lift && request.Driver {
			roadPlan, err = m.router.LiftPlan(tx, []matching.Request{request})
			if err != nil {
				return nil, err
			}
		}
		role := "passenger"
		if request.Driver {
			role = "rider"
		}
		members = append(members, bson.M{"user_id": userID, "role": role, "readiness_confirmed": false, "joined_at": now})
		status := "pending"
		if len(members) > 1 {
			status = "confirming"
		}
		if roadPlan != nil {
			departure = roadPlan.Earliest
			end = roadPlan.Latest
		}
		deadline = minTime(deadline, end)
		fields := bson.M{"members": members, "status": status, "female_only": femaleOnly, "mode": assignedMode,
			"route_search": SearchRegionForRide(ride), "reminder_schedule": nil, "road_plan": roadPlan, "departure_start": departure, "departure_end": end, "expires_at": end,
			"confirmation_deadline": deadline, "finalized_at": nil, "updated_at": now}
		if targetGroup == "" {
			fields["group_id"] = groupID
			fields["date"] = ride.Date
			fields["time_window"] = ride.Window
			fields["timezone_offset_minutes"] = ride.Offset
			fields["created_at"] = now
			fields["origin_user_id"] = userID
			fields["search_expires_at"] = ride.Expires
			fields["route_summary"] = ride.PickupLabel + " → " + ride.DropLabel
			fields["pickup_summary"] = ride.PickupLabel
			fields["drop_summary"] = ride.DropLabel
			if _, err = groups.InsertOne(tx, fields); err != nil {
				return nil, err
			}
		} else {
			// Writing the group document serializes simultaneous joins under Mongo's
			// transaction conflict/retry rules; seats are re-evaluated on each retry.
			changed, updateErr := groups.UpdateOne(tx, bson.M{"group_id": groupID, "status": bson.M{"$in": []string{"pending", "confirming"}}}, bson.M{"$set": fields})
			if updateErr != nil {
				return nil, updateErr
			}
			if changed.MatchedCount != 1 {
				return nil, ErrGroupUnavailable
			}
		}
		assignmentFilter := bson.M{"request_id": requestID, "user_id": userID, "status": bson.M{"$in": []string{"pending", "matching"}}, "expires_at": bson.M{"$gt": now}}
		if sourceGroup != "" {
			assignmentFilter["status"] = "matched"
			assignmentFilter["group_id"] = sourceGroup
			cancelled, cancelErr := groups.UpdateOne(tx, bson.M{"group_id": sourceGroup, "status": "pending", "members.user_id": userID}, bson.M{"$set": bson.M{"status": "cancelled", "merged_into": groupID, "origin_user_id": userID, "members": bson.A{}, "road_plan": nil, "updated_at": now}})
			if cancelErr != nil {
				return nil, cancelErr
			}
			if cancelled.MatchedCount != 1 {
				return nil, ErrGroupUnavailable
			}
		}
		changed, err := m.rides().UpdateOne(tx, assignmentFilter, bson.M{"$set": bson.M{"status": "matched", "group_id": groupID, "matched_at": now}})
		if err != nil {
			return nil, err
		}
		if changed.ModifiedCount != 1 {
			return nil, ErrGroupUnavailable
		}
		// The event commits with membership; a later worker can deliver/retry it.
		_, err = m.collection.Database().Collection("notification_outbox").InsertOne(tx, bson.M{
			"type": "group_updated", "group_id": groupID, "user_id": userID, "created_at": now,
			"status": "pending", "attempts": 0,
		})
		if err != nil {
			return nil, err
		}
		return groupID, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return "", err
	}
	id, ok := result.(string)
	if !ok {
		return "", errors.New("assignment returned no group")
	}
	return id, nil
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func (a *AuthDependencies) assignGroup(w http.ResponseWriter, r *http.Request, user User) {
	if _, ok := a.rideStorage(w, user); !ok {
		return
	}
	var input struct {
		GroupID string `json:"group_id"`
	}
	if decodeBody(r, &input) != nil {
		writeJSON(w, 422, map[string]string{"detail": "Provide an optional group_id"})
		return
	}
	repository, ok := a.Users.(AssignmentRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Group storage is not configured"})
		return
	}
	id, err := repository.AssignGroup(r.Context(), user.ID, r.PathValue("request_id"), input.GroupID)
	if errors.Is(err, ErrGroupUnavailable) || errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 409, map[string]string{"detail": "Request or group is no longer available"})
		return
	}
	if errors.Is(err, ErrRoadRoutingRequired) || errors.Is(err, ErrNoRoadPlan) {
		writeJSON(w, 409, map[string]string{"detail": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to reserve group membership"})
		return
	}
	writeJSON(w, 200, map[string]string{"group_id": id, "message": "Group membership reserved"})
}
