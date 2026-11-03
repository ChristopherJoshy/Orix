package server

import (
	"context"
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func validRideInput() RideInput {
	offset := 330
	return RideInput{PickupLat: 10, PickupLng: 76, DropLat: 10.1, DropLng: 76, PickupLabel: "Gate", DropLabel: "Station", Date: "2026-11-03", Start: "08:00", End: "09:00", Offset: &offset}
}
func TestRideValidation(t *testing.T) {
	now := time.Date(2026, 11, 3, 1, 0, 0, 0, time.UTC)
	user := User{ID: "u"}
	tests := []struct {
		name   string
		change func(*RideInput)
		valid  bool
	}{
		{"valid", func(*RideInput) {}, true},
		{"invalid coordinate", func(r *RideInput) { r.PickupLat = 91 }, false},
		{"missing offset", func(r *RideInput) { r.Offset = nil }, false},
		{"invalid date", func(r *RideInput) { r.Date = "2026-02-30" }, false},
		{"invalid time", func(r *RideInput) { r.Start = "25:00" }, false},
		{"reversed window", func(r *RideInput) { r.End = "07:00" }, false},
		{"expired", func(r *RideInput) { r.Date = "2026-11-02" }, false},
		{"invalid mode", func(r *RideInput) { r.Mode = "taxi" }, false},
		{"gender from trusted profile", func(r *RideInput) { r.FemaleOnly = true }, false},
		{"zero distance", func(r *RideInput) { r.DropLat = r.PickupLat }, false},
		{"lift capacity", func(r *RideInput) { r.Mode = "lift"; r.Driver = true; r.Seats = 0 }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validRideInput()
			tt.change(&input)
			ride, err := buildRide(input, user, now)
			if (err == nil) != tt.valid {
				t.Fatal(err)
			}
			if tt.valid && (ride.StartUTC.Hour() != 2 || ride.StartUTC.Minute() != 30 || ride.UserID != "u" || ride.Mode != "auto" || ride.Pickup.Coordinates[0] != 76) {
				t.Fatalf("bad normalization: %+v", ride)
			}
		})
	}
}
func TestCoordinatesRequiredAndIdentityNotAccepted(t *testing.T) {
	for _, payload := range []string{`{}`, `{"pickup_lat":null,"pickup_lng":0,"drop_lat":1,"drop_lng":0}`, `{"pickup_lat":0,"pickup_lng":0,"drop_lat":1,"drop_lng":0,"user_id":"other"}`} {
		var input RideInput
		if json.Unmarshal([]byte(payload), &input) == nil {
			t.Fatalf("accepted %s", payload)
		}
	}
}

type ownedRideRepository struct {
	fakeUsers
	ride             Ride
	problem          error
	owner, requested string
}

func (f *ownedRideRepository) CreateRide(context.Context, Ride) error { return nil }
func (f *ownedRideRepository) ListRides(context.Context, string, bool) ([]Ride, error) {
	return nil, nil
}
func (f *ownedRideRepository) CancelRide(context.Context, string, string) (bool, error) {
	return false, nil
}
func (f *ownedRideRepository) OwnedRide(_ context.Context, owner, id string) (Ride, error) {
	f.owner = owner
	f.requested = id
	return f.ride, f.problem
}
func (f *ownedRideRepository) CandidateGroups(context.Context, Ride) ([]CandidateGroup, error) {
	return nil, nil
}
func (f *ownedRideRepository) GroupForMember(context.Context, string, string) (CandidateGroup, error) {
	return CandidateGroup{}, nil
}
func TestGetRideScopesHistoricalReadsToAuthenticatedOwner(t *testing.T) {
	for _, test := range []struct {
		name      string
		problem   error
		onboarded bool
		want      int
	}{
		{"historical", nil, true, 200},
		{"missing or another owner", mongo.ErrNoDocuments, true, 404},
		{"storage unavailable", errors.New("private storage error"), true, 503},
		{"onboarding required", nil, false, 403},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &ownedRideRepository{ride: Ride{ID: "past", Status: "completed"}, problem: test.problem}
			auth := AuthDependencies{Users: repository}
			request := httptest.NewRequest("GET", "/api/v1/rides/past?user_id=someone-else", nil)
			request.SetPathValue("request_id", "past")
			response := httptest.NewRecorder()
			auth.getRide(response, request, User{ID: "authenticated-owner", Onboarded: test.onboarded})
			if response.Code != test.want {
				t.Fatalf("status %d, want %d", response.Code, test.want)
			}
			if test.onboarded && (repository.owner != "authenticated-owner" || repository.requested != "past") {
				t.Fatal("read did not use authenticated ownership")
			}
			if strings.Contains(response.Body.String(), "private storage error") {
				t.Fatal("storage details exposed")
			}
		})
	}
}
