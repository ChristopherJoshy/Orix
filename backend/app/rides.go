package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"orix/backend/internal/matching"
)

type RideInput struct {
	PickupPlaceID string        `json:"pickup_place_id"`
	DropPlaceID   string        `json:"drop_place_id"`
	PickupLat     float64       `json:"pickup_lat"`
	PickupLng     float64       `json:"pickup_lng"`
	DropLat       float64       `json:"drop_lat"`
	DropLng       float64       `json:"drop_lng"`
	PickupLabel   string        `json:"pickup_label"`
	DropLabel     string        `json:"drop_label"`
	Date          string        `json:"date"`
	Start         string        `json:"time_window_start"`
	End           string        `json:"time_window_end"`
	Offset        *int          `json:"timezone_offset_minutes"`
	AllowRiders   *bool         `json:"allow_riders"`
	FemaleOnly    bool          `json:"female_only"`
	Mode          matching.Mode `json:"mode"`
	Driver        bool          `json:"driver"`
	Seats         int           `json:"seats"`
}

func (input *RideInput) UnmarshalJSON(data []byte) error {
	type plain RideInput
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, key := range []string{"pickup_lat", "pickup_lng", "drop_lat", "drop_lng"} {
		value, ok := fields[key]
		if !ok || string(value) == "null" {
			return errors.New("coordinates are required")
		}
	}
	// Preserve strict field checking even through this custom decoder.
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	return decoder.Decode((*plain)(input))
}

type RideWindow struct {
	Start string `json:"start" bson:"start"`
	End   string `json:"end" bson:"end"`
}
type GeoPoint struct {
	Type        string    `json:"type" bson:"type"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates"`
}
type Ride struct {
	RejectedGroups []string      `json:"-" bson:"rejected_group_ids,omitempty"`
	RejectedUsers  []string      `json:"-" bson:"rejected_user_ids,omitempty"`
	PickupPlaceID  string        `json:"pickup_place_id" bson:"pickup_place_id"`
	DropPlaceID    string        `json:"drop_place_id" bson:"drop_place_id"`
	CandidateAfter string        `json:"-" bson:"candidate_scan_after,omitempty"`
	ID             string        `json:"request_id" bson:"request_id"`
	UserID         string        `json:"-" bson:"user_id"`
	PickupLabel    string        `json:"pickup_label" bson:"pickup_label"`
	DropLabel      string        `json:"drop_label" bson:"drop_label"`
	Pickup         GeoPoint      `json:"pickup" bson:"pickup"`
	Drop           GeoPoint      `json:"drop" bson:"drop"`
	Date           string        `json:"date" bson:"date"`
	Window         RideWindow    `json:"time_window" bson:"time_window"`
	Offset         int           `json:"timezone_offset_minutes" bson:"timezone_offset_minutes"`
	AllowRiders    bool          `json:"allow_riders" bson:"allow_riders"`
	FemaleOnly     bool          `json:"female_only" bson:"female_only"`
	Mode           matching.Mode `json:"mode" bson:"mode"`
	Driver         bool          `json:"driver" bson:"driver"`
	Seats          int           `json:"seats" bson:"seats"`
	Status         string        `json:"status" bson:"status"`
	GroupID        *string       `json:"group_id" bson:"group_id"`
	Created        time.Time     `json:"created_at" bson:"created_at"`
	Expires        time.Time     `json:"expires_at" bson:"expires_at"`
	StartUTC       time.Time     `json:"departure_start" bson:"departure_start"`
	EndUTC         time.Time     `json:"departure_end" bson:"departure_end"`
	Remaining      int64         `json:"time_remaining_seconds" bson:"-"`
}

var ErrActiveRide = errors.New("active ride already exists")

type RideRepository interface {
	CreateRide(context.Context, Ride) error
	ListRides(context.Context, string, bool) ([]Ride, error)
	CancelRide(context.Context, string, string) (bool, error)
}

func (m mongoUsers) rides() *mongo.Collection {
	return m.collection.Database().Collection("ride_requests")
}
func (m mongoUsers) CreateRide(ctx context.Context, ride Ride) error {
	// Release expired pending slots before the unique active-user index is checked.
	_, err := m.rides().UpdateMany(ctx, bson.M{"user_id": ride.UserID, "status": bson.M{"$in": []string{"pending", "matching"}}, "expires_at": bson.M{"$lte": ride.Created}}, bson.M{"$set": bson.M{"status": "expired"}})
	if err != nil {
		return err
	}
	_, err = m.rides().InsertOne(ctx, ride)
	if mongo.IsDuplicateKeyError(err) {
		return ErrActiveRide
	}
	return err
}
func (m mongoUsers) ListRides(ctx context.Context, user string, history bool) ([]Ride, error) {
	query := bson.M{"user_id": user}
	if !history {
		query["$or"] = bson.A{
			bson.M{"status": "matched"},
			bson.M{"status": bson.M{"$in": []string{"pending", "matching"}}, "expires_at": bson.M{"$gt": time.Now().UTC()}},
		}
	}
	pageQuery(ctx, query, "created_at", "request_id")
	cursor, err := m.rides().Find(ctx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "request_id", Value: -1}}).SetLimit(pageLimit(ctx, 100)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rides := make([]Ride, 0)
	err = cursor.All(ctx, &rides)
	if err == nil {
		rides = trimPage(ctx, rides, func(ride Ride) (time.Time, string) { return ride.Created, ride.ID })
	}
	return rides, err
}
func (m mongoUsers) CancelRide(ctx context.Context, user, id string) (bool, error) {
	result, err := m.rides().UpdateOne(ctx, bson.M{"user_id": user, "request_id": id, "status": bson.M{"$in": []string{"pending", "matching"}}}, bson.M{"$set": bson.M{"status": "cancelled"}})
	if err != nil {
		return false, err
	}
	return result.ModifiedCount == 1, nil
}
func buildRide(input RideInput, user User, now time.Time) (Ride, error) {
	invalid := errors.New("Provide valid locations, travel mode and a future time window")
	if input.Offset == nil || *input.Offset < -720 || *input.Offset > 840 {
		return Ride{}, invalid
	}
	for _, label := range []string{input.PickupLabel, input.DropLabel} {
		if strings.TrimSpace(label) == "" || len([]rune(label)) > 200 {
			return Ride{}, invalid
		}
	}
	zone := time.FixedZone("ride", *input.Offset*60)
	start, err := time.ParseInLocation("2006-01-02 15:04", input.Date+" "+input.Start, zone)
	if err != nil {
		return Ride{}, invalid
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", input.Date+" "+input.End, zone)
	if err != nil || !end.After(start) || !end.After(now) || start.After(now.Add(90*24*time.Hour)) {
		return Ride{}, invalid
	}
	if len(input.PickupPlaceID) > 600 || len(input.DropPlaceID) > 600 {
		return Ride{}, invalid
	}
	mode := input.Mode
	if mode == "" {
		mode = matching.Auto
	}
	female := user.Gender != nil && (*user.Gender == "F" || strings.EqualFold(*user.Gender, "female"))
	request := matching.Request{ID: "validation", UserID: user.ID, Pickup: matching.Point{Lat: input.PickupLat, Lng: input.PickupLng}, Drop: matching.Point{Lat: input.DropLat, Lng: input.DropLng}, Start: start, End: end, Mode: mode, Driver: input.Driver, Seats: input.Seats, FemaleOnly: input.FemaleOnly, Female: female}
	if request.Validate() != nil {
		return Ride{}, invalid
	}
	raw := make([]byte, 16)
	if _, err = rand.Read(raw); err != nil {
		return Ride{}, err
	}
	expires := now.Add(15 * time.Hour)
	if end.Before(expires) {
		expires = end
	}
	allow := true
	if input.AllowRiders != nil {
		allow = *input.AllowRiders
	}
	return Ride{PickupPlaceID: input.PickupPlaceID, DropPlaceID: input.DropPlaceID, ID: hex.EncodeToString(raw), UserID: user.ID, PickupLabel: strings.TrimSpace(input.PickupLabel), DropLabel: strings.TrimSpace(input.DropLabel), Pickup: GeoPoint{"Point", []float64{input.PickupLng, input.PickupLat}}, Drop: GeoPoint{"Point", []float64{input.DropLng, input.DropLat}}, Date: input.Date, Window: RideWindow{input.Start, input.End}, Offset: *input.Offset, AllowRiders: allow, FemaleOnly: input.FemaleOnly, Mode: mode, Driver: input.Driver, Seats: input.Seats, Status: "pending", Created: now.UTC(), Expires: expires.UTC(), StartUTC: start.UTC(), EndUTC: end.UTC()}, nil
}
func rideResponse(ride Ride) Ride {
	ride.Remaining = max(0, int64(time.Until(ride.Expires).Seconds()))
	return ride
}
func (a *AuthDependencies) rideStorage(w http.ResponseWriter, user User) (RideRepository, bool) {
	if !user.Onboarded {
		writeJSON(w, 403, map[string]any{"detail": map[string]string{"error": "onboarding_required", "message": "Please complete onboarding first."}})
		return nil, false
	}
	repository, ok := a.Users.(RideRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Ride storage is not configured"})
	}
	return repository, ok
}
func (a *AuthDependencies) createRide(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.rideStorage(w, user)
	if !ok {
		return
	}
	var input RideInput
	if decodeBody(r, &input) != nil {
		writeJSON(w, 422, map[string]string{"detail": "Invalid ride fields"})
		return
	}
	ride, err := buildRide(input, user, time.Now().UTC())
	if err != nil {
		writeJSON(w, 422, map[string]string{"detail": err.Error()})
		return
	}
	if err = repository.CreateRide(r.Context(), ride); err != nil {
		if errors.Is(err, ErrActiveRide) {
			writeJSON(w, 409, map[string]string{"detail": "You already have an active ride request"})
		} else {
			writeJSON(w, 503, map[string]string{"detail": "Unable to create your ride"})
		}
		return
	}
	writeJSON(w, 201, rideResponse(ride))
}
func (a *AuthDependencies) listRides(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.rideStorage(w, user)
	if !ok {
		return
	}
	paged, valid := preparePage(w, r, "rides:"+user.ID, 100)
	if !valid {
		return
	}
	r = paged
	rides, err := repository.ListRides(r.Context(), user.ID, r.URL.Query().Get("include_history") == "true")
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load rides"})
		return
	}
	if rides == nil {
		rides = []Ride{}
	}
	for i := range rides {
		rides[i] = rideResponse(rides[i])
	}
	writePageHeader(w, r)
	writeJSON(w, 200, rides)
}
func (a *AuthDependencies) cancelRide(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.rideStorage(w, user)
	if !ok {
		return
	}
	cancelled, err := repository.CancelRide(r.Context(), user.ID, r.PathValue("request_id"))
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to cancel ride"})
		return
	}
	if !cancelled {
		writeJSON(w, 404, map[string]string{"detail": "Ride request not found or cannot be cancelled"})
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Ride request cancelled"})
}

// getRide returns only the authenticated owner's request, including historical routes.
func (a *AuthDependencies) getRide(w http.ResponseWriter, r *http.Request, user User) {
	if _, ok := a.rideStorage(w, user); !ok {
		return
	}
	repository, ok := a.Users.(MatchRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Ride storage is not configured"})
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
	writeJSON(w, 200, rideResponse(ride))
}
