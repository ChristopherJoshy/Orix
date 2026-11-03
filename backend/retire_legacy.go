package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	server "orix/backend/app"
)

// retireLegacyData prepares the final post-test data retirement. It never removes
// source files, backups, Redis namespaces or records from active app collections.
func retireLegacyData() error {
	database, project := os.Getenv("MONGODB_DATABASE"), os.Getenv("FIREBASE_PROJECT_ID")
	actorUID := os.Getenv("RETIREMENT_ADMIN_FIREBASE_UID")
	if database == "" || project == "" || os.Getenv("MONGODB_URI") == "" || os.Getenv("RETIREMENT_DATABASE") != database || actorUID == "" || len(actorUID) > 128 {
		return errors.New("retirement requires an explicit Firebase project, administrator UID, Mongo URI and matching RETIREMENT_DATABASE")
	}
	switch os.Getenv("APP_ENV") {
	case "local":
		if err := validateLocalEmulator(); err != nil {
			return err
		}
	case "staging", "production":
		if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != "" {
			return errors.New("cloud retirement cannot use Auth Emulator")
		}
	default:
		return errors.New("select APP_ENV=local, staging or production")
	}
	apply := false
	switch os.Getenv("RETIREMENT_APPLY") {
	case "", "false":
	case "true":
		apply = true
	default:
		return errors.New("RETIREMENT_APPLY must be true or false")
	}
	if apply && (os.Getenv("RETIREMENT_WRITERS_STOPPED") != "true" || os.Getenv("RETIREMENT_TRANSITION_TESTED") != "true" || os.Getenv("RETIREMENT_TEST_REFERENCE") == "") {
		return errors.New("apply requires stopped legacy/Go writers, tested transition and an explicit test evidence reference")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	_, identity, err := server.OpenFirebase(ctx, project, os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"))
	if err != nil {
		return err
	}
	actor, err := identity.GetUser(ctx, actorUID)
	if err != nil || actor.Disabled || !actor.EmailVerified || actor.Email == "" {
		return errors.New("retirement requires an enabled verified administrator identity")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		return err
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = client.Disconnect(cleanup)
	}()
	plan, retireErr := server.RetireLegacyData(ctx, client.Database(database), actorUID, os.Getenv("RETIREMENT_TEST_REFERENCE"), apply)
	// Report incremental progress even when a later batch fails. No secret values
	// or retired document contents are included in the reviewable plan/results.
	if err := json.NewEncoder(os.Stdout).Encode(plan); err != nil {
		return err
	}
	return retireErr
}
