package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	server "orix/backend/app"
)

func bootstrapAdmin() error {
	database, project := os.Getenv("MONGODB_DATABASE"), os.Getenv("FIREBASE_PROJECT_ID")
	uid := os.Getenv("BOOTSTRAP_ADMIN_FIREBASE_UID")
	if uid == "" || len(uid) > 128 || database == "" || project == "" || os.Getenv("ADMIN_DATABASE") != database || os.Getenv("BOOTSTRAP_APPLY") != "true" || len(os.Getenv("ADMIN_API_SECRET")) < 32 {
		return errors.New("bootstrap requires an existing Firebase UID, explicit database/apply selection and a 32-character admin secret")
	}
	switch os.Getenv("APP_ENV") {
	case "local":
		if err := validateLocalEmulator(); err != nil {
			return err
		}
	case "staging", "production":
		if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != "" {
			return errors.New("cloud bootstrap cannot use an authentication emulator")
		}
	default:
		return errors.New("select APP_ENV=local, staging or production")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, identity, err := server.OpenFirebase(ctx, project, os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"))
	if err != nil {
		return err
	}
	firebaseUser, err := identity.GetUser(ctx, uid)
	if err != nil {
		return errors.New("selected Firebase user could not be verified")
	}
	if firebaseUser.Disabled || !firebaseUser.EmailVerified || firebaseUser.Email == "" {
		return errors.New("administrator identity must have a verified email and be enabled")
	}
	dependencies, closeAuth, err := server.OpenAuth(ctx, project, os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"), os.Getenv("MONGODB_URI"), database)
	if err != nil {
		return err
	}
	defer closeAuth()
	var photo *string
	if firebaseUser.PhotoURL != "" {
		photo = &firebaseUser.PhotoURL
	}
	user, _, err := dependencies.Users.GetOrCreate(ctx, server.Identity{UID: uid, Email: firebaseUser.Email, Name: firebaseUser.DisplayName, Photo: photo})
	if err != nil {
		return err
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
	session, err := client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := client.Database(database)
		var current server.User
		if err := db.Collection("users").FindOne(tx, bson.M{"firebase_uid": uid, "user_id": user.ID}).Decode(&current); err != nil {
			return nil, err
		}
		if current.Status != "active" {
			return nil, errors.New("administrator app account must be active")
		}
		var guard struct {
			UID string `bson:"firebase_uid"`
		}
		lookup := db.Collection("app_config").FindOne(tx, bson.M{"_id": "admin_bootstrap"}).Decode(&guard)
		if lookup != nil && !errors.Is(lookup, mongo.ErrNoDocuments) {
			return nil, lookup
		}
		if lookup == nil && guard.UID != uid {
			return nil, errors.New("first administrator bootstrap is already assigned")
		}
		if current.Role == "admin" {
			return nil, nil
		}
		count, err := db.Collection("users").CountDocuments(tx, bson.M{"role": "admin"})
		if err != nil {
			return nil, err
		}
		if count != 0 {
			return nil, errors.New("an administrator already exists; use authenticated staff management")
		}
		now := time.Now().UTC()
		if _, err = db.Collection("app_config").UpdateOne(tx, bson.M{"_id": "admin_bootstrap"}, bson.M{"$setOnInsert": bson.M{"firebase_uid": uid, "created_at": now}}, options.UpdateOne().SetUpsert(true)); err != nil {
			return nil, err
		}
		if _, err = db.Collection("users").UpdateOne(tx, bson.M{"user_id": user.ID, "status": "active"}, bson.M{"$set": bson.M{"role": "admin", "updated_at": now}}); err != nil {
			return nil, err
		}
		_, err = server.WriteAudit(tx, db, bson.M{"actor_id": user.ID, "action": "bootstrap_admin", "target_id": user.ID, "created_at": now})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return err
	}
	fmt.Printf("Administrator ready: %s\n", user.ID)
	return nil
}
