package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"orix/backend/internal/matching"
)

// A worker carries its scan position independently of public list pagination.
// The next position reflects scanned groups, including incompatible ones, so an
// old, unsuitable batch cannot permanently hide later candidates.
type candidateScan struct{ After, Next string }
type candidateScanKey struct{}

type CandidateGroup struct {
	ID          string `bson:"group_id"`
	PickupLabel string `bson:"pickup_summary"`
	DropLabel   string `bson:"drop_summary"`
	Status      string `bson:"status"`
	Members     []struct {
		UserID    string `bson:"user_id"`
		Confirmed bool   `bson:"readiness_confirmed"`
	} `bson:"members"`
	Requests []matching.Request `bson:"-"`
}
type MatchRepository interface {
	OwnedRide(context.Context, string, string) (Ride, error)
	CandidateGroups(context.Context, Ride) ([]CandidateGroup, error)
	GroupForMember(context.Context, string, string) (CandidateGroup, error)
}

func (m mongoUsers) OwnedRide(ctx context.Context, user, id string) (Ride, error) {
	var ride Ride
	err := m.rides().FindOne(ctx, bson.M{"request_id": id, "user_id": user}).Decode(&ride)
	return ride, err
}
func (m mongoUsers) GroupForMember(ctx context.Context, user, id string) (CandidateGroup, error) {
	var group CandidateGroup
	err := m.collection.Database().Collection("ride_groups").FindOne(ctx, bson.M{"group_id": id, "members.user_id": user}).Decode(&group)
	return group, err
}

func routeRequest(ride Ride, user User) (matching.Request, error) {
	if len(ride.Pickup.Coordinates) != 2 || len(ride.Drop.Coordinates) != 2 {
		return matching.Request{}, errors.New("missing route coordinates")
	}
	start, end := ride.StartUTC, ride.EndUTC
	if start.IsZero() || end.IsZero() {
		zone := time.FixedZone("ride", ride.Offset*60)
		var err error
		start, err = time.ParseInLocation("2006-01-02 15:04", ride.Date+" "+ride.Window.Start, zone)
		if err != nil {
			return matching.Request{}, err
		}
		end, err = time.ParseInLocation("2006-01-02 15:04", ride.Date+" "+ride.Window.End, zone)
		if err != nil {
			return matching.Request{}, err
		}
	}
	mode := ride.Mode
	if mode == "" {
		mode = matching.Auto
	}
	request := matching.Request{IgnoredGroups: ride.RejectedGroups, IgnoredUsers: ride.RejectedUsers, DestinationID: ride.DropPlaceID, AllowLift: ride.AllowRiders, ID: ride.ID, UserID: ride.UserID, Pickup: matching.Point{Lat: ride.Pickup.Coordinates[1], Lng: ride.Pickup.Coordinates[0]}, Drop: matching.Point{Lat: ride.Drop.Coordinates[1], Lng: ride.Drop.Coordinates[0]}, Start: start, End: end, Mode: mode, Driver: ride.Driver, Seats: ride.Seats, FemaleOnly: ride.FemaleOnly, Female: user.Gender != nil && *user.Gender == "F"}
	return request, request.Validate()
}

func (m mongoUsers) CandidateGroups(ctx context.Context, ride Ride) ([]CandidateGroup, error) {
	// Query a calendar envelope to include legacy groups without UTC fields.
	day, err := time.Parse("2006-01-02", ride.Date)
	if err != nil {
		return nil, err
	}
	start := day.Add(-48 * time.Hour).Format("2006-01-02")
	end := day.Add(48 * time.Hour).Format("2006-01-02")
	db := m.collection.Database()
	mode := ride.Mode
	if mode == "" {
		mode = matching.Auto
	}
	query := bson.M{
		"status":          bson.M{"$in": []string{"pending", "confirming"}},
		"date":            bson.M{"$gte": start, "$lte": end},
		"members.user_id": bson.M{"$ne": ride.UserID},
		"members.0":       bson.M{"$exists": true},
		"members.3":       bson.M{"$exists": false},
	}
	conditions := bson.A{}
	if region := candidateRegionFilter(ride); region != nil {
		conditions = append(conditions, region)
	}
	// Generic passenger requests may join any compatible pool. Explicit legacy
	// modes remain a constraint; Rank checks every persisted member before commit.
	allowed := bson.A{matching.Auto, matching.Partner, matching.Pool, ""}
	if mode == matching.Lift || (mode == matching.Pool && ride.AllowRiders) {
		allowed = append(allowed, matching.Lift)
	}
	if mode != matching.Pool && mode != matching.Lift {
		allowed = bson.A{mode, matching.Pool, matching.Auto, ""}
	}
	conditions = append(conditions, bson.M{"$or": bson.A{bson.M{"mode": bson.M{"$in": allowed}}, bson.M{"mode": bson.M{"$exists": false}}}})

	now := time.Now().UTC()
	// Lift windows describe pickups at different stops, so their literal times
	// cannot be intersected here. The road planner checks the actual itinerary.
	if mode != matching.Lift && !(mode == matching.Pool && ride.AllowRiders) && !ride.EndUTC.IsZero() {
		conditions = append(conditions,
			bson.M{"$or": bson.A{bson.M{"departure_start": bson.M{"$lte": ride.EndUTC}}, bson.M{"departure_start": bson.M{"$exists": false}}}},
			bson.M{"$or": bson.A{bson.M{"departure_end": bson.M{"$gt": now, "$gte": ride.StartUTC}}, bson.M{"departure_end": bson.M{"$exists": false}}}},
		)
	}
	if len(conditions) > 0 {
		query["$and"] = conditions
	}
	scan, _ := ctx.Value(candidateScanKey{}).(*candidateScan)
	if scan != nil && scan.After != "" {
		query["group_id"] = bson.M{"$gt": scan.After}
	}
	const batchSize = 200
	cursor, err := db.Collection("ride_groups").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "group_id", Value: 1}}).SetLimit(batchSize+1).SetProjection(bson.M{
		"_id": 0, "group_id": 1, "pickup_summary": 1, "drop_summary": 1, "status": 1,
		"members.user_id": 1, "members.readiness_confirmed": 1,
	}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	groups := []CandidateGroup{}
	if err = cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	if scan != nil {
		scan.Next = ""
	}
	if len(groups) > batchSize {
		groups = groups[:batchSize]
		if scan != nil {
			scan.Next = groups[len(groups)-1].ID
		}
	}
	if len(groups) == 0 {
		return groups, nil
	}
	ids := make([]string, 0, len(groups))
	users := []string{}
	seenUsers := map[string]bool{}
	for _, group := range groups {
		ids = append(ids, group.ID)
		for _, member := range group.Members {
			if !seenUsers[member.UserID] {
				users = append(users, member.UserID)
				seenUsers[member.UserID] = true
			}
		}
	}
	// Fetch member routes and profiles in batches, not once per candidate/member.
	routesCursor, err := m.rides().Find(ctx, bson.M{"group_id": bson.M{"$in": ids}, "status": "matched"}, options.Find().SetLimit(int64(len(groups)*matching.DefaultPolicy().MaxMembers+1)).SetProjection(bson.M{
		"_id": 0, "request_id": 1, "user_id": 1, "pickup": 1, "drop": 1, "rejected_group_ids": 1, "rejected_user_ids": 1, "pickup_place_id": 1, "drop_place_id": 1, "date": 1, "time_window": 1,
		"timezone_offset_minutes": 1, "mode": 1, "driver": 1, "seats": 1, "female_only": 1, "allow_riders": 1,
		"departure_start": 1, "departure_end": 1, "group_id": 1, "status": 1,
	}))
	if err != nil {
		return nil, err
	}
	defer routesCursor.Close(ctx)
	var routes []Ride
	if err = routesCursor.All(ctx, &routes); err != nil {
		return nil, err
	}
	if len(routes) > len(groups)*matching.DefaultPolicy().MaxMembers {
		return nil, errors.New("candidate member routes exceed the group bound")
	}
	userCursor, err := m.collection.Find(ctx, bson.M{"user_id": bson.M{"$in": users}}, options.Find().SetProjection(bson.M{
		"_id": 0, "user_id": 1, "gender": 1, "status": 1, "suspension_expires_at": 1, "onboarding_completed": 1,
	}))
	if err != nil {
		return nil, err
	}
	defer userCursor.Close(ctx)
	var profiles []User
	if err = userCursor.All(ctx, &profiles); err != nil {
		return nil, err
	}
	byUser := map[string]User{}
	for _, profile := range profiles {
		if profile.Onboarded && !accountRestricted(profile) {
			byUser[profile.ID] = profile
		}
	}
	type key struct{ group, user string }
	byMember := map[key][]Ride{}
	for _, route := range routes {
		if route.GroupID != nil {
			k := key{*route.GroupID, route.UserID}
			byMember[k] = append(byMember[k], route)
		}
	}
	blocked, err := m.BlockedPeers(ctx, ride.UserID)
	if err != nil {
		return nil, err
	}
	result := []CandidateGroup{}
	for _, group := range groups {
		if len(group.Members) == 0 || len(group.Members) >= matching.DefaultPolicy().MaxMembers {
			continue
		}
		valid := true
		for _, member := range group.Members {
			profile, exists := byUser[member.UserID]
			memberRoutes := byMember[key{group.ID, member.UserID}]
			if !exists || blocked[member.UserID] || len(memberRoutes) != 1 {
				valid = false
				break
			}
			request, err := routeRequest(memberRoutes[0], profile)
			if err != nil {
				valid = false
				break
			}
			group.Requests = append(group.Requests, request)
		}
		if valid {
			result = append(result, group)
		}
	}
	return result, nil
}

type RouteSuggestion struct {
	GroupID                  string    `json:"group_id"`
	PickupLabel              string    `json:"pickup_label"`
	DropLabel                string    `json:"drop_label"`
	Members                  int       `json:"member_count"`
	Score                    float64   `json:"match_score"`
	Departure                time.Time `json:"departure"`
	RoadVerificationRequired bool      `json:"road_verification_required"`
}

func rankGroups(ride Ride, user User, groups []CandidateGroup, now time.Time) ([]RouteSuggestion, error) {
	request, err := routeRequest(ride, user)
	if err != nil {
		return nil, err
	}
	if request.Start.Before(now) {
		request.Start = now
	}
	candidates := make([]matching.Group, 0, len(groups))
	byID := map[string]CandidateGroup{}
	for _, group := range groups {
		members := append([]matching.Request(nil), group.Requests...)
		valid := true
		for i := range members {
			if !members[i].End.After(now) {
				valid = false
				break
			}
			if members[i].Start.Before(now) {
				members[i].Start = now
			}
		}
		if !valid {
			continue
		}
		candidates = append(candidates, matching.Group{ID: group.ID, Members: members})
		byID[group.ID] = group
	}
	ranked, err := matching.Rank(request, candidates, matching.DefaultPolicy())
	if err != nil {
		return nil, err
	}
	results := make([]RouteSuggestion, 0, min(len(ranked), 20))
	for _, match := range ranked {
		if len(results) == 20 {
			break
		}
		group := byID[match.GroupID]
		results = append(results, RouteSuggestion{match.GroupID, group.PickupLabel, group.DropLabel, len(group.Members), match.Score, match.Departure, matching.RequiresLift(request, group.Requests)})
	}
	return results, nil
}
func (a *AuthDependencies) suggestions(w http.ResponseWriter, r *http.Request, user User) {
	if _, ok := a.rideStorage(w, user); !ok {
		return
	}
	repository, ok := a.Users.(MatchRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Matching storage is not configured"})
		return
	}
	ride, err := repository.OwnedRide(r.Context(), user.ID, r.PathValue("request_id"))
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Ride request not found"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load ride"})
		return
	}
	searching := ride.Status == "pending" || ride.Status == "matching"
	if ride.Status == "matched" && ride.GroupID != nil {
		group, err := repository.GroupForMember(r.Context(), user.ID, *ride.GroupID)
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			writeJSON(w, 503, map[string]string{"detail": "Unable to load the waiting group"})
			return
		}
		searching = err == nil && group.Status == "pending" && len(group.Members) == 1 && group.Members[0].UserID == user.ID && !group.Members[0].Confirmed
	}
	if !searching || !ride.Expires.After(time.Now()) {
		writeJSON(w, 409, map[string]string{"detail": "This request is no longer searching"})
		return
	}
	groups, err := repository.CandidateGroups(r.Context(), ride)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to find route suggestions"})
		return
	}
	results, err := rankGroups(ride, user, groups, time.Now().UTC())
	if err != nil {
		writeJSON(w, 422, map[string]string{"detail": "Ride route needs updating"})
		return
	}
	writeJSON(w, 200, map[string]any{"suggestions": results})
}
func (a *AuthDependencies) matchingStatus(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.rideStorage(w, user)
	if !ok {
		return
	}
	rides, err := repository.ListRides(r.Context(), user.ID, false)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load matching status"})
		return
	}
	if len(rides) == 0 {
		writeJSON(w, 200, map[string]bool{"has_active_request": false})
		return
	}
	ride := rideResponse(rides[0])
	response := map[string]any{"request": ride, "group_id": ride.GroupID, "has_active_request": true, "request_id": ride.ID, "status": ride.Status, "pickup_label": ride.PickupLabel, "drop_label": ride.DropLabel, "date": ride.Date, "time_window": ride.Window, "expires_at": ride.Expires, "time_remaining_seconds": ride.Remaining, "my_readiness_confirmed": false, "pending_confirmation_group_id": nil}
	if ride.GroupID != nil {
		matches, ok := a.Users.(MatchRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Group storage is not configured"})
			return
		}
		group, err := matches.GroupForMember(r.Context(), user.ID, *ride.GroupID)
		if err != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to load group status"})
			return
		}
		response["group_status"] = group.Status
		response["group_size"] = len(group.Members)
		if group.Status == "confirming" {
			response["pending_confirmation_group_id"] = group.ID
		}
		for _, member := range group.Members {
			if member.UserID == user.ID {
				response["my_readiness_confirmed"] = member.Confirmed
			}
		}
	}
	writeJSON(w, 200, response)
}
