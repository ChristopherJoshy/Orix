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
	"orix/backend/internal/matching"
)

type RideEditor interface {
	EditRide(context.Context, string, string, RideInput) (Ride, error)
}

func (m mongoUsers) EditRide(ctx context.Context, user, id string, input RideInput) (Ride, error) {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return Ride{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		state, err := m.Maintenance(tx)
		if err != nil {
			return nil, err
		}
		if state.Enabled {
			return nil, ErrGroupUnavailable
		}
		var owner User
		if err = m.collection.FindOne(tx, bson.M{"user_id": user}).Decode(&owner); err != nil {
			return nil, err
		}
		if accountRestricted(owner) || !owner.Onboarded {
			return nil, ErrGroupUnavailable
		}
		// Serialize edits against safety/identity changes and concurrent reservations.
		if _, err = m.collection.UpdateOne(tx, bson.M{"user_id": user}, bson.M{"$inc": bson.M{"ride_edit_version": 1}}); err != nil {
			return nil, err
		}
		original, err := m.OwnedRide(tx, user, id)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		replacement, err := buildRide(input, owner, now)
		if err != nil {
			return nil, err
		}
		replacement.RejectedGroups = original.RejectedGroups
		replacement.RejectedUsers = original.RejectedUsers
		replacement.ID = original.ID
		replacement.Created = original.Created
		if !original.Expires.After(now) {
			return nil, ErrGroupUnavailable
		}
		if original.Status == "matched" && original.GroupID != nil {
			var group GroupRecord
			groups := m.collection.Database().Collection("ride_groups")
			if err = groups.FindOne(tx, bson.M{"group_id": *original.GroupID, "status": "pending", "members.user_id": user}).Decode(&group); err != nil {
				return nil, ErrGroupUnavailable
			}
			if len(group.Members) != 1 || group.Members[0].UserID != user || group.Members[0].Ready {
				return nil, ErrGroupUnavailable
			}
			var plan *RoadPlan
			if replacement.Mode == matching.Lift && replacement.Driver {
				route, routeErr := routeRequest(replacement, owner)
				if routeErr != nil {
					return nil, routeErr
				}
				plan, err = m.router.LiftPlan(tx, []matching.Request{route})
				if err != nil {
					return nil, err
				}
			}
			departure, end := replacement.StartUTC, replacement.EndUTC
			if plan != nil {
				departure, end = plan.Earliest, plan.Latest
			}
			deadline := minTime(now.Add(time.Hour), end)
			role := "passenger"
			if replacement.Driver {
				role = "rider"
			}
			group.Members[0].Role = role
			group.Members[0].Here = false
			group.Members[0].Finished = false
			fields := bson.M{"date": replacement.Date, "time_window": replacement.Window, "timezone_offset_minutes": replacement.Offset,
				"mode": matching.AssignedMode(matching.Request{Mode: replacement.Mode}, nil), "female_only": replacement.FemaleOnly, "route_search": SearchRegionForRide(replacement), "road_plan": plan, "reminder_schedule": nil,
				"route_summary": replacement.PickupLabel + " → " + replacement.DropLabel, "pickup_summary": replacement.PickupLabel, "drop_summary": replacement.DropLabel,
				"departure_start": departure, "departure_end": end, "expires_at": end, "search_expires_at": replacement.Expires,
				"confirmation_deadline": deadline, "finalized_at": nil, "members": group.Members, "next_match_at": now, "updated_at": now}
			if _, err = groups.UpdateOne(tx, bson.M{"group_id": group.ID, "status": "pending"}, bson.M{"$set": fields, "$unset": bson.M{"match_lease": "", "match_lease_until": ""}}); err != nil {
				return nil, err
			}
			replacement.Status = "matched"
			replacement.GroupID = original.GroupID
			if _, err = m.collection.Database().Collection("notification_outbox").InsertOne(tx, bson.M{"type": "group_updated", "group_id": group.ID, "user_id": user, "status": "pending", "attempts": 0, "created_at": now}); err != nil {
				return nil, err
			}
		} else if original.Status != "pending" && original.Status != "matching" {
			return nil, ErrGroupUnavailable
		}
		encoded, err := bson.Marshal(replacement)
		if err != nil {
			return nil, err
		}
		var fields bson.M
		if err = bson.Unmarshal(encoded, &fields); err != nil {
			return nil, err
		}
		fields["updated_at"] = now
		fields["next_match_at"] = now
		filter := bson.M{"request_id": id, "user_id": user, "status": original.Status}
		if original.GroupID != nil {
			filter["group_id"] = *original.GroupID
		}
		changed, err := m.rides().UpdateOne(tx, filter, bson.M{"$set": fields, "$unset": bson.M{"match_lease": "", "match_lease_until": "", "match_error": "", "candidate_scan_after": ""}})
		if err != nil {
			return nil, err
		}
		if changed.MatchedCount != 1 {
			return nil, ErrGroupUnavailable
		}
		return replacement, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return Ride{}, err
	}
	ride, ok := result.(Ride)
	if !ok {
		return Ride{}, errors.New("missing updated ride")
	}
	return ride, nil
}
func (a *AuthDependencies) editRide(w http.ResponseWriter, r *http.Request, user User) {
	if _, ok := a.rideStorage(w, user); !ok {
		return
	}
	var input RideInput
	if decodeBody(r, &input) != nil {
		writeJSON(w, 422, map[string]string{"detail": "Invalid ride fields"})
		return
	}
	// Validate early; the transaction validates again against the current profile.
	if _, err := buildRide(input, user, time.Now().UTC()); err != nil {
		writeJSON(w, 422, map[string]string{"detail": "Provide a valid route and future time window"})
		return
	}
	repository, ok := a.Users.(RideEditor)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Ride editing unavailable"})
		return
	}
	ride, err := repository.EditRide(r.Context(), user.ID, r.PathValue("request_id"), input)
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Ride request not found"})
		return
	}
	if errors.Is(err, ErrGroupUnavailable) {
		writeJSON(w, 409, map[string]string{"detail": "Only an unmatched request or unconfirmed one-person group can be edited. Leave a shared group before creating a different trip."})
		return
	}
	if errors.Is(err, ErrNoRoadPlan) || errors.Is(err, ErrRoadRoutingRequired) {
		writeJSON(w, 409, map[string]string{"detail": "A verified road route is required for this lift"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to update your ride"})
		return
	}
	writeJSON(w, 200, rideResponse(ride))
}
