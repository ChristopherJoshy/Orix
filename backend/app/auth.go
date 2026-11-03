package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/api/option"
)

type Identity struct {
	UID, Email, Name string
	Photo            *string
}
type User struct {
	Occupation        string         `json:"occupation" bson:"occupation"`
	Role              string         `json:"-" bson:"role"`
	Phone             *string        `json:"-" bson:"phone"`
	Gender            *string        `json:"-" bson:"gender"`
	IsRider           bool           `json:"-" bson:"is_rider"`
	RiderInfo         map[string]any `json:"-" bson:"rider_info"`
	ID                string         `json:"user_id" bson:"user_id"`
	Email             string         `json:"email" bson:"email"`
	Name              string         `json:"display_name" bson:"display_name"`
	Photo             *string        `json:"photo_url" bson:"photo_url"`
	Onboarded         bool           `json:"onboarding_completed" bson:"onboarding_completed"`
	Status            string         `json:"-" bson:"status"`
	SuspensionExpires *time.Time     `json:"-" bson:"suspension_expires_at"`
}
type TokenVerifier interface {
	Verify(context.Context, string) (Identity, error)
}
type UserRepository interface {
	GetOrCreate(context.Context, Identity) (User, bool, error)
}
type AuthDependencies struct {
	Verifier TokenVerifier
	Users    UserRepository
}

type firebaseVerifier struct {
	client                *auth.Client
	allowEmulatorPassword bool
}

func (v firebaseVerifier) Verify(ctx context.Context, raw string) (Identity, error) {
	token, err := v.client.VerifyIDTokenAndCheckRevoked(ctx, raw)
	if err != nil {
		return Identity{}, err
	}
	firebaseClaims, _ := token.Claims["firebase"].(map[string]any)
	provider, _ := firebaseClaims["sign_in_provider"].(string)
	if provider != "google.com" && !(v.allowEmulatorPassword && provider == "password") {
		return Identity{}, errors.New("Google sign-in required")
	}
	email, _ := token.Claims["email"].(string)
	name, _ := token.Claims["name"].(string)
	verified, _ := token.Claims["email_verified"].(bool)
	if token.UID == "" || email == "" || !verified {
		return Identity{}, errors.New("verified email required")
	}
	var photo *string
	if value, ok := token.Claims["picture"].(string); ok {
		photo = &value
	}
	return Identity{token.UID, email, name, photo}, nil
}

type mongoUsers struct {
	collection *mongo.Collection
	push       *messaging.Client
	identity   *auth.Client
	router     *RoadRouter
}

func (m mongoUsers) GetOrCreate(ctx context.Context, identity Identity) (User, bool, error) {
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return User{}, false, err
	}
	id := hex.EncodeToString(idBytes)
	name := identity.Name
	if name == "" {
		name = strings.Split(identity.Email, "@")[0]
	}
	now := time.Now().UTC()
	insert := bson.M{"user_id": id, "firebase_uid": identity.UID, "email": identity.Email,
		"display_name": name, "photo_url": identity.Photo, "onboarding_completed": false,
		"safety_consent": false, "status": "active", "role": "user", "trust_score": .5,
		"created_at": now, "updated_at": now}
	var user User
	err := m.collection.FindOneAndUpdate(ctx, bson.M{"firebase_uid": identity.UID}, bson.M{"$setOnInsert": insert}, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&user)
	if mongo.IsDuplicateKeyError(err) {
		err = m.collection.FindOne(ctx, bson.M{"firebase_uid": identity.UID}).Decode(&user)
	}
	return user, user.ID == id, err
}

// OpenFirebase shares credential parsing across the runtime and guarded tools.
func OpenFirebase(ctx context.Context, project, credentialsPath, credentialsJSON string) (*firebase.App, *auth.Client, error) {
	if project == "" {
		return nil, nil, errors.New("Firebase project is required")
	}
	var opts []option.ClientOption
	if credentialsJSON != "" {
		credentialBytes := []byte(credentialsJSON)
		if !json.Valid(credentialBytes) {
			decoded, decodeErr := base64.StdEncoding.DecodeString(credentialsJSON)
			if decodeErr != nil || !json.Valid(decoded) {
				return nil, nil, errors.New("Firebase credentials must be JSON or base64 JSON")
			}
			credentialBytes = decoded
		}
		opts = append(opts, option.WithCredentialsJSON(credentialBytes))
	} else if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: project}, opts...)
	if err != nil {
		return nil, nil, err
	}
	verifier, err := app.Auth(ctx)
	if err != nil {
		return nil, nil, err
	}
	return app, verifier, nil
}

// OpenAuth initializes Firebase authentication and a transactional MongoDB deployment.
// The explicitly local emulator configuration is validated by the command entry point.
func OpenAuth(ctx context.Context, project, credentialsPath, credentialsJSON, uri, database string) (*AuthDependencies, func(), error) {
	if project == "" || uri == "" || database == "" {
		return nil, nil, errors.New("Firebase project and MongoDB settings are required")
	}
	app, verifier, err := OpenFirebase(ctx, project, credentialsPath, credentialsJSON)
	if err != nil {
		return nil, nil, err
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}
	closeClient := func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(shutdown)
	}
	if err = requireTransactionalMongo(ctx, client); err != nil {
		closeClient()
		return nil, nil, err
	}
	collection := client.Database(database).Collection("users")
	_, err = collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "notification_preferences.promotions", Value: 1}, {Key: "onboarding_completed", Value: 1}, {Key: "last_engagement_at", Value: 1}}, Options: options.Index().SetName("engagement_eligible_v1")},
		{Keys: bson.D{{Key: "firebase_uid", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		closeClient()
		return nil, nil, err
	}
	rideIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "request_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetName("one_active_ride_per_user").SetUnique(true).SetPartialFilterExpression(bson.M{"status": bson.M{"$in": []string{"pending", "matching", "matched"}}})},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err = client.Database(database).Collection("ride_requests").Indexes().CreateMany(ctx, rideIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	chatIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "message_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "dedupe_key", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"dedupe_key": bson.M{"$exists": true}})},
		{Keys: bson.D{{Key: "group_id", Value: 1}, {Key: "created_at", Value: -1}, {Key: "message_id", Value: -1}}},
	}
	if _, err = client.Database(database).Collection("chat_messages").Indexes().CreateMany(ctx, chatIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	notificationIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "notification_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "read", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err = client.Database(database).Collection("notifications").Indexes().CreateMany(ctx, notificationIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	if _, err = client.Database(database).Collection("notification_outbox").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "priority", Value: -1}, {Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}},
		{Keys: bson.D{{Key: "type", Value: 1}, {Key: "status", Value: 1}, {Key: "priority", Value: -1}, {Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "next_attempt_at", Value: 1}, {Key: "created_at", Value: 1}}},
	}); err != nil {
		closeClient()
		return nil, nil, err
	}
	reportIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "report_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "reporter_user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err = client.Database(database).Collection("reports").Indexes().CreateMany(ctx, reportIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	blockIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "blocker_user_id", Value: 1}, {Key: "blocked_user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "blocked_user_id", Value: 1}, {Key: "blocker_user_id", Value: 1}}},
	}
	if _, err = client.Database(database).Collection("user_blocks").Indexes().CreateMany(ctx, blockIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	supportIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "ticket_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "create_retry_key", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"create_retry_key": bson.M{"$type": "string"}})},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "updated_at", Value: -1}}},
	}
	if _, err = client.Database(database).Collection("support_tickets").Indexes().CreateMany(ctx, supportIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	ratingIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "rating_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "group_id", Value: 1}, {Key: "rater_user_id", Value: 1}, {Key: "rated_user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "rated_user_id", Value: 1}, {Key: "rating", Value: 1}}},
	}
	if _, err = client.Database(database).Collection("ratings").Indexes().CreateMany(ctx, ratingIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	deletionIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "next_attempt_at", Value: 1}}},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	}
	if _, err = client.Database(database).Collection("account_deletions").Indexes().CreateMany(ctx, deletionIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	matchingIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "next_match_at", Value: 1}, {Key: "created_at", Value: 1}}},
	}
	if _, err = client.Database(database).Collection("ride_requests").Indexes().CreateMany(ctx, matchingIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	groupIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "reminder_schedule.ride_time", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "mode", Value: 1}, {Key: "route_search.version", Value: 1}, {Key: "route_search.min_lat", Value: 1}, {Key: "route_search.max_lat", Value: 1}, {Key: "route_search.min_lng", Value: 1}, {Key: "route_search.max_lng", Value: 1}}, Options: options.Index().SetName("candidate_route_bounds_v1")},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "mode", Value: 1}, {Key: "group_id", Value: 1}, {Key: "date", Value: 1}}},
		{Keys: bson.D{{Key: "group_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "date", Value: 1}, {Key: "created_at", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "next_match_at", Value: 1}, {Key: "created_at", Value: 1}}},
		{Keys: bson.D{{Key: "members.user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "departure_start", Value: 1}}},
	}
	if _, err = client.Database(database).Collection("ride_groups").Indexes().CreateMany(ctx, groupIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	lifecycleIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "search_expires_at", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "confirmation_deadline", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "departure_end", Value: 1}}},
	}
	if _, err = client.Database(database).Collection("ride_groups").Indexes().CreateMany(ctx, lifecycleIndexes); err != nil {
		closeClient()
		return nil, nil, err
	}
	if _, err = client.Database(database).Collection("user_avatars").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)}); err != nil {
		closeClient()
		return nil, nil, err
	}
	paginationIndexes := map[string][]mongo.IndexModel{
		"admin_logs": {
			{Keys: bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}},
			{Keys: bson.D{{Key: "actor_id", Value: 1}, {Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}},
			{Keys: bson.D{{Key: "target_id", Value: 1}, {Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}},
		},
		"ride_requests": {{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}, {Key: "request_id", Value: -1}}}},
		"ride_groups":   {{Keys: bson.D{{Key: "members.user_id", Value: 1}, {Key: "created_at", Value: -1}, {Key: "group_id", Value: -1}}}},
		"notifications": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}, {Key: "notification_id", Value: -1}}},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "read", Value: 1}, {Key: "created_at", Value: -1}, {Key: "notification_id", Value: -1}}},
		},
		"reports": {
			{Keys: bson.D{{Key: "reporter_user_id", Value: 1}, {Key: "created_at", Value: -1}, {Key: "report_id", Value: -1}}},
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}, {Key: "report_id", Value: -1}}},
		},
		"support_tickets": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "updated_at", Value: -1}, {Key: "ticket_id", Value: -1}}},
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "updated_at", Value: -1}, {Key: "ticket_id", Value: -1}}},
		},
	}
	for name, indexes := range paginationIndexes {
		if _, err = client.Database(database).Collection(name).Indexes().CreateMany(ctx, indexes); err != nil {
			closeClient()
			return nil, nil, err
		}
	}
	var push *messaging.Client
	if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") == "" {
		push, err = app.Messaging(ctx)
		if err != nil {
			closeClient()
			return nil, nil, err
		}
	}
	var router *RoadRouter
	if os.Getenv("ROUTING_PROVIDER") == "geoapify" {
		router, err = NewGeoapifyRouter(os.Getenv("GEOCODING_API_TOKEN"))
	} else {
		router, err = NewRoadRouter(os.Getenv("ROUTING_BASE_URL"))
	}
	if err != nil {
		closeClient()
		return nil, nil, err
	}
	return &AuthDependencies{firebaseVerifier{client: verifier, allowEmulatorPassword: project == "demo-orix" && os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != ""}, mongoUsers{collection: collection, push: push, identity: verifier, router: router}}, closeClient, nil
}

func (a *AuthDependencies) verify(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.Verifier == nil || a.Users == nil {
		writeJSON(w, 503, map[string]string{"detail": "Authentication is not configured"})
		return
	}
	var input struct {
		Token string `json:"id_token"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || len(input.Token) == 0 || len(input.Token) > 16384 {
		writeJSON(w, 422, map[string]string{"detail": "A valid id_token is required"})
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeJSON(w, 422, map[string]string{"detail": "Expected one JSON object"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	identity, err := a.Verifier.Verify(ctx, input.Token)
	if err != nil {
		writeJSON(w, 401, map[string]string{"detail": "Invalid or expired token"})
		return
	}
	user, created, err := a.Users.GetOrCreate(ctx, identity)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load your account. Please try again."})
		return
	}
	if accountRestricted(user) {
		writeJSON(w, 403, map[string]any{"detail": map[string]string{"error": "account_" + user.Status, "message": "Your account is " + user.Status + "."}})
		return
	}
	writeJSON(w, 200, struct {
		User
		IsNew bool   `json:"is_new_user"`
		Role  string `json:"role"`
	}{user, created, user.Role})
}
