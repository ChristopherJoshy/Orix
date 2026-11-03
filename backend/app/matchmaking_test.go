package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"orix/backend/internal/matching"
)

func TestLegacyRouteNormalization(t *testing.T) {
	input := validRideInput()
	now := time.Date(2026, 11, 3, 1, 0, 0, 0, time.UTC)
	ride, err := buildRide(input, User{ID: "owner"}, now)
	if err != nil {
		t.Fatal(err)
	}
	ride.StartUTC = time.Time{}
	ride.EndUTC = time.Time{}
	ride.Mode = ""
	request, err := routeRequest(ride, User{})
	if err != nil || request.Start.UTC().Hour() != 2 || request.Mode != matching.Auto {
		t.Fatal(request, err)
	}
	ride.Pickup.Coordinates = nil
	if _, err = routeRequest(ride, User{}); err == nil {
		t.Fatal("missing coordinates accepted")
	}
}
func TestSuggestionsCheckEveryMemberAndHidePrivateData(t *testing.T) {
	now := time.Date(2026, 11, 3, 1, 0, 0, 0, time.UTC)
	ride, _ := buildRide(validRideInput(), User{ID: "owner"}, now)
	member, _ := routeRequest(ride, User{})
	member.ID = "member-request"
	member.UserID = "private-user"
	member.Pickup.Lat += .001 // Within the new 300 m pickup limit; same destination.
	group := CandidateGroup{ID: "group", PickupLabel: "Meeting point", DropLabel: "Station", Requests: []matching.Request{member}}
	group.Members = append(group.Members, struct {
		UserID    string `bson:"user_id"`
		Confirmed bool   `bson:"readiness_confirmed"`
	}{UserID: member.UserID})
	results, err := rankGroups(ride, User{}, []CandidateGroup{group}, now)
	if err != nil || len(results) != 1 {
		t.Fatal(results, err)
	}
	data, _ := json.Marshal(results)
	if strings.Contains(string(data), "private-user") || strings.Contains(string(data), "coordinates") {
		t.Fatal("private data leaked")
	}
	opposite := member
	opposite.ID = "other"
	opposite.UserID = "other"
	opposite.Pickup, opposite.Drop = opposite.Drop, opposite.Pickup
	group.Requests = append(group.Requests, opposite)
	results, err = rankGroups(ride, User{}, []CandidateGroup{group}, now)
	if err != nil || len(results) != 0 {
		t.Fatal("incompatible member matched")
	}
}
func TestSuggestionsNeverChoosePastDeparture(t *testing.T) {
	now := time.Date(2026, 11, 3, 2, 45, 0, 0, time.UTC)
	ride, _ := buildRide(validRideInput(), User{ID: "owner"}, now)
	member, _ := routeRequest(ride, User{})
	member.ID = "other"
	member.UserID = "other"
	results, err := rankGroups(ride, User{}, []CandidateGroup{{ID: "g", Requests: []matching.Request{member}}}, now)
	if err != nil || len(results) != 1 || results[0].Departure.Before(now) {
		t.Fatal(results, err)
	}
}
