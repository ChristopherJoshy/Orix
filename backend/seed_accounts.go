package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	server "orix/backend/app"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

func validateLocalEmulator() error {
	host, _, err := net.SplitHostPort(os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"))
	if err != nil || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
		return errors.New("authentication emulator must be a loopback host:port")
	}
	if os.Getenv("APP_ENV") != "local" || os.Getenv("FIREBASE_PROJECT_ID") != "demo-orix" {
		return errors.New("emulator mode requires APP_ENV=local and FIREBASE_PROJECT_ID=demo-orix")
	}
	mongoURL, parseErr := url.Parse(os.Getenv("MONGODB_URI"))
	if parseErr != nil || mongoURL.Scheme != "mongodb" || (mongoURL.Hostname() != "127.0.0.1" && mongoURL.Hostname() != "localhost" && mongoURL.Hostname() != "::1") || os.Getenv("MONGODB_DATABASE") != "orix_internal_test" {
		return errors.New("local testing requires loopback MongoDB and MONGODB_DATABASE=orix_internal_test")
	}
	return nil
}

var testPersonas = []struct{ slug, name, gender string }{
	{"traveler-a", "Traveler A", "F"}, {"traveler-b", "Traveler B", "F"},
	{"traveler-c", "Traveler C", "M"}, {"driver", "Lift Driver", "M"},
	{"safety", "Safety Traveler", "F"}, {"opposite", "Incompatible Traveler", "M"},
	{"admin", "Moderation Admin", "F"}, {"newcomer", "New Traveler", "M"},
}

// Travel profiles use the real onboarding API. Only the isolated admin fixture
// receives a privileged role, after normal Firebase authentication and onboarding.
func seedTestAccounts() error {
	if err := validateLocalEmulator(); err != nil {
		return err
	}
	password := os.Getenv("TEST_ACCOUNT_PASSWORD")
	if len(password) < 12 {
		return errors.New("set TEST_ACCOUNT_PASSWORD to at least 12 characters")
	}
	origin := os.Getenv("TEST_API_BASE_URL")
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" || u.User != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" && u.Hostname() != "::1") || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("TEST_API_BASE_URL must be a local HTTP origin")
	}
	origin = strings.TrimRight(origin, "/")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: "demo-orix"})
	if err != nil {
		return err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for i, p := range testPersonas {
		uid := "orix-test-" + p.slug
		email := p.slug + "@orix.example"
		existing, lookupErr := client.GetUser(ctx, uid)
		if auth.IsUserNotFound(lookupErr) {
			_, err = client.CreateUser(ctx, (&auth.UserToCreate{}).UID(uid).Email(email).EmailVerified(true).Password(password).DisplayName(p.name))
		} else if lookupErr != nil {
			return fmt.Errorf("lookup %s failed", p.slug)
		} else if existing.Email != email {
			return fmt.Errorf("existing %s has an unexpected email", uid)
		} else {
			_, err = client.UpdateUser(ctx, uid, (&auth.UserToUpdate{}).Password(password).EmailVerified(true).DisplayName(p.name))
		}
		if err != nil {
			return fmt.Errorf("provision %s failed", p.slug)
		}
		var session struct {
			Token string `json:"idToken"`
		}
		endpoint := "http://" + os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") + "/identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=local-emulator"
		if err = postSeedJSON(ctx, httpClient, endpoint, "", map[string]any{"email": email, "password": password, "returnSecureToken": true}, &session); err != nil {
			return fmt.Errorf("sign in %s: %w", p.slug, err)
		}
		if session.Token == "" {
			return errors.New("emulator returned no identity token")
		}
		if err = postSeedJSON(ctx, httpClient, origin+"/api/v1/auth/verify", "", map[string]string{"id_token": session.Token}, nil); err != nil {
			return fmt.Errorf("verify %s: %w", p.slug, err)
		}
		if p.slug == "newcomer" {
			fmt.Printf("Ready for onboarding: %s (%s); existing onboarding state is preserved\n", p.name, email)
			continue
		}
		payload := map[string]any{"phone": fmt.Sprintf("+15550101%03d", i+1), "gender": p.gender, "safety_consent": true, "privacy_consent": true}
		if err = postSeedJSON(ctx, httpClient, origin+"/api/v1/auth/onboard", session.Token, payload, nil); err != nil {
			return fmt.Errorf("onboard %s: %w", p.slug, err)
		}
		if p.slug == "admin" {
			if err = provisionLocalAdmin(ctx, uid); err != nil {
				return err
			}
		}
		fmt.Printf("Ready: %s (%s)\n", p.name, email)
	}
	return nil
}
func postSeedJSON(ctx context.Context, client *http.Client, endpoint, token string, payload, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return errors.New("local service is unreachable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("local service returned HTTP %d", response.StatusCode)
	}
	if result != nil {
		return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(result)
	}
	return nil
}

// provisionLocalAdmin cannot target a supplied UID or a cloud database. Keeping
// this outside the HTTP API prevents a client-controlled testing role bypass.
func provisionLocalAdmin(ctx context.Context, uid string) error {
	if err := validateLocalEmulator(); err != nil {
		return err
	}
	if uid != "orix-test-admin" {
		return errors.New("unexpected internal admin identity")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(cleanup)
	}()
	session, err := client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	db := client.Database("orix_internal_test")
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		var profile struct {
			ID   string `bson:"user_id"`
			Role string `bson:"role"`
		}
		filter := bson.M{"firebase_uid": uid, "email": "admin@orix.example", "status": "active", "onboarding_completed": true}
		if err := db.Collection("users").FindOne(tx, filter).Decode(&profile); err != nil {
			return nil, err
		}
		if profile.Role == "admin" {
			return nil, nil
		}
		result, err := db.Collection("users").UpdateOne(tx, filter, bson.M{"$set": bson.M{"role": "admin", "updated_at": time.Now().UTC()}})
		if err != nil {
			return nil, err
		}
		if result.MatchedCount != 1 {
			return nil, errors.New("local admin profile changed during provisioning")
		}
		_, err = server.WriteAudit(tx, db, bson.M{"actor_id": profile.ID, "target_id": profile.ID, "action": "seed_local_test_admin", "created_at": time.Now().UTC()})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}
