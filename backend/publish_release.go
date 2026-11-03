package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	server "orix/backend/app"
)

// Operator-only command: authenticated SSH and the server's private credentials
// are its trust boundary. It does not mint user tokens, change roles or add an
// HTTP authentication exception. Publication still checks the admin in its transaction.
func publishAndroid() error {
	database := os.Getenv("MONGODB_DATABASE")
	path, version, digest := os.Getenv("RELEASE_APK_PATH"), os.Getenv("RELEASE_VERSION"), strings.ToLower(os.Getenv("RELEASE_SHA256"))
	if err := validateReleaseSelection(database, path, version, digest); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return errors.New("release APK could not be opened")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > 256<<20 {
		return errors.New("release APK must be a regular file of at most 256 MB")
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil || hex.EncodeToString(hash.Sum(nil)) != digest {
		return errors.New("release APK checksum does not match the explicit selection")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return errors.New("release APK could not be read")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		return errors.New("release storage unavailable")
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = client.Disconnect(cleanup)
	}()
	db := client.Database(database)
	// Use the existing guarded owner identity; never select an arbitrary account or create one.
	var guard struct {
		UID string `bson:"firebase_uid"`
	}
	if db.Collection("app_config").FindOne(ctx, bson.M{"_id": "admin_bootstrap"}).Decode(&guard) != nil || guard.UID == "" {
		return errors.New("the verified owner administrator must be configured first")
	}
	var actor server.User
	if db.Collection("users").FindOne(ctx, bson.M{"firebase_uid": guard.UID, "role": "admin", "status": "active"}).Decode(&actor) != nil {
		return errors.New("owner administrator is not active")
	}
	_, identity, err := server.OpenFirebase(ctx, os.Getenv("FIREBASE_PROJECT_ID"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"))
	if err != nil {
		return errors.New("release identity verification unavailable")
	}
	account, err := identity.GetUser(ctx, guard.UID)
	if err != nil || account.Disabled || !account.EmailVerified {
		return errors.New("owner administrator Firebase identity is not enabled and verified")
	}
	dependencies, closeAuth, err := server.OpenAuth(ctx, os.Getenv("FIREBASE_PROJECT_ID"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"), os.Getenv("MONGODB_URI"), database)
	if err != nil {
		return errors.New("release service unavailable")
	}
	defer closeAuth()
	repo, ok := dependencies.Users.(server.DownloadRepository)
	if !ok {
		return errors.New("release publication unavailable")
	}
	// An operator audit distinguishes this authenticated deployment action from a UI upload.
	if _, err = server.WriteAudit(ctx, db, bson.M{"actor_id": actor.ID, "action": "operator_android_publication_requested", "source": "authenticated_server_operator", "version": version, "sha256": digest}); err != nil {
		return errors.New("release operator audit unavailable")
	}
	published, err := repo.PublicSettings(ctx)
	if err != nil {
		return errors.New("current release metadata could not be read")
	}
	// Retrying after an uncertain response must not upload the same artifact again.
	if published.APKVersion != version || published.APKDigest != digest || published.APKSize != info.Size() || published.APKID == nil {
		ticket, err := repo.CreateAPKTicket(ctx, actor.ID, version, digest, info.Size())
		if err != nil {
			return errors.New("release upload authorization failed")
		}
		published, err = repo.PublishAPK(ctx, actor.ID, ticket, file)
		if err != nil {
			return errors.New("release publication failed; inspect its safe stage diagnostic")
		}
	}
	if published.APKVersion != version || published.APKDigest != digest || published.APKSize != info.Size() || published.APKID == nil {
		return errors.New("published release metadata does not match the selected APK")
	}
	versions, ok := dependencies.Users.(server.VersionRepository)
	if !ok {
		return errors.New("APK published; version metadata requires administrator review")
	}
	current, err := versions.AppVersion(ctx)
	if err != nil {
		return errors.New("APK published; version metadata could not be read")
	}
	current.Current = version
	title, notes := "A fresh version of ORIX", "Refined pixel layouts, optional donations and Manglish reminders."
	current.Title = &title
	current.Notes = &notes
	// Preserve minimum version and official store URLs; do not force an update.
	if !server.ValidAppVersionConfig(current) {
		return errors.New("APK published; version configuration requires administrator review")
	}
	if err = versions.SetAppVersion(ctx, actor.ID, current); err != nil {
		return errors.New("APK published; current version metadata could not be updated")
	}
	fmt.Printf("Published Android %s; SHA-256 %s; %d bytes\n", version, digest, info.Size())
	return nil
}
func validateReleaseSelection(database, path, version, digest string) error {
	if os.Getenv("RELEASE_APPLY") != "true" || database == "" || os.Getenv("RELEASE_DATABASE") != database || os.Getenv("APP_ENV") != "production" || os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != "" || len(os.Getenv("ADMIN_API_SECRET")) < 32 {
		return errors.New("release publication requires explicit production database/apply selection and private server credentials")
	}
	decoded, err := hex.DecodeString(digest)
	if path == "" || !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(version) || !server.ValidAppVersionConfig(server.AppVersion{Current: version, Minimum: version}) || err != nil || len(decoded) != 32 {
		return errors.New("select the APK path, stable version and exact SHA-256")
	}
	return nil
}
