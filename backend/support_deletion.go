package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"firebase.google.com/go/v4/auth"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	server "orix/backend/app"
)

// supportAccountDeletion is deliberately CLI-only. Server credentials plus an
// existing enabled administrator and completed ownership review are required;
// merely knowing an email/UID never grants a public deletion capability.
func supportAccountDeletion() error {
	database, project := os.Getenv("MONGODB_DATABASE"), os.Getenv("FIREBASE_PROJECT_ID")
	if database == "" || project == "" || os.Getenv("MONGODB_URI") == "" || os.Getenv("DELETION_DATABASE") != database {
		return errors.New("support deletion requires an explicit Firebase project, Mongo URI and matching DELETION_DATABASE")
	}
	switch os.Getenv("APP_ENV") {
	case "local":
		if err := validateLocalEmulator(); err != nil {
			return err
		}
	case "staging", "production":
		if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != "" {
			return errors.New("cloud support deletion cannot use Auth Emulator")
		}
	default:
		return errors.New("select APP_ENV=local, staging or production")
	}
	apply := false
	switch os.Getenv("DELETION_APPLY") {
	case "", "false":
	case "true":
		apply = true
	default:
		return errors.New("DELETION_APPLY must be true or false")
	}
	if apply && os.Getenv("DELETION_OWNERSHIP_VERIFIED") != "true" {
		return errors.New("apply requires completed ownership review and DELETION_OWNERSHIP_VERIFIED=true")
	}
	input := server.SupportDeletionRequest{
		ActorFirebaseUID:  os.Getenv("DELETION_ADMIN_FIREBASE_UID"),
		TargetFirebaseUID: os.Getenv("DELETION_TARGET_FIREBASE_UID"),
		CaseID:            os.Getenv("DELETION_CASE_ID"),
	}
	// Reject missing identifiers before SDK initialization or any network requests.
	if err := input.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, identity, err := server.OpenFirebase(ctx, project, os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"))
	if err != nil {
		return err
	}
	actor, err := identity.GetUser(ctx, input.ActorFirebaseUID)
	if err != nil || actor.Disabled || !actor.EmailVerified || actor.Email == "" {
		return errors.New("selected administrator must have an enabled, verified Firebase identity")
	}
	// Disabled target identities are supported. Missing identities can still have
	// app records needing removal; permission/network errors are never ignored.
	if _, err := identity.GetUser(ctx, input.TargetFirebaseUID); err != nil && !auth.IsUserNotFound(err) {
		return errors.New("target identity lookup failed")
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
	plan, err := server.ReviewSupportDeletion(ctx, client.Database(database), input, apply)
	if err != nil {
		return err
	}
	// Identifiers/status only; never print email, proof documents, tokens or secrets.
	return json.NewEncoder(os.Stdout).Encode(plan)
}
