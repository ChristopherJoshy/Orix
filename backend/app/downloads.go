package server

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const maxAPKSize int64 = 256 << 20

var errInvalidAPK = errors.New("invalid APK upload")

type PublicSettings struct {
	AndroidPrompt bool           `json:"android_prompt" bson:"android_prompt"`
	PlayStoreURL  string         `json:"play_store_url" bson:"play_store_url"`
	SupportEmail  string         `json:"support_email" bson:"support_email"`
	APKVersion    string         `json:"apk_version" bson:"apk_version"`
	APKDigest     string         `json:"apk_sha256" bson:"apk_sha256"`
	APKSize       int64          `json:"apk_size" bson:"apk_size"`
	APKID         *bson.ObjectID `json:"-" bson:"apk_id,omitempty"`
}
type DownloadRepository interface {
	PublicSettings(context.Context) (PublicSettings, error)
	SetPublicSettings(context.Context, string, PublicSettings) error
	CreateAPKTicket(context.Context, string, string, string, int64) (string, error)
	PublishAPK(context.Context, string, string, io.Reader) (PublicSettings, error)
	OpenAPK(context.Context) (*mongo.GridFSDownloadStream, PublicSettings, error)
}

func (m mongoUsers) PublicSettings(ctx context.Context) (PublicSettings, error) {
	value := PublicSettings{AndroidPrompt: true, SupportEmail: os.Getenv("ACCOUNT_SUPPORT_EMAIL")}
	err := m.collection.Database().Collection("app_config").FindOne(ctx, bson.M{"_id": "public"}).Decode(&value)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return value, nil
	}
	return value, err
}
func (m mongoUsers) SetPublicSettings(ctx context.Context, actor string, value PublicSettings) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		_, err := m.collection.Database().Collection("app_config").UpdateOne(tx, bson.M{"_id": "public"}, bson.M{"$set": bson.M{"android_prompt": value.AndroidPrompt, "play_store_url": value.PlayStoreURL, "support_email": value.SupportEmail}}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return nil, err
		}
		_, err = WriteAudit(tx, m.collection.Database(), bson.M{"actor_id": actor, "action": "updated_public_settings", "created_at": time.Now().UTC()})
		return nil, err
	})
	return err
}
func (m mongoUsers) CreateAPKTicket(ctx context.Context, actor, version, digest string, size int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	ticket := hex.EncodeToString(raw)
	// Short-lived, single-use grant scoped to one authenticated administrator/file.
	_, err := m.collection.Database().Collection("apk_upload_tickets").InsertOne(ctx, bson.M{"_id": stableID(ticket), "actor_id": actor, "version": version, "sha256": digest, "size": size, "expires_at": time.Now().UTC().Add(5 * time.Minute)})
	return ticket, err
}

var apkUploadSlots = make(chan struct{}, 2)

func (m mongoUsers) PublishAPK(ctx context.Context, actor, ticket string, source io.Reader) (result PublicSettings, resultErr error) {
	stage := "acquire_upload_slot"
	defer func() {
		if resultErr != nil {
			slog.Error("APK publication failed", "stage", stage)
		}
	}()
	select {
	case apkUploadSlots <- struct{}{}:
		defer func() { <-apkUploadSlots }()
	case <-ctx.Done():
		return PublicSettings{}, ctx.Err()
	}
	stage = "validate_upload_grant"
	var grant struct {
		Version string `bson:"version"`
		Digest  string `bson:"sha256"`
		Size    int64  `bson:"size"`
	}
	// Consume before receiving bytes; failed transfers require a fresh grant.
	err := m.collection.Database().Collection("apk_upload_tickets").FindOneAndDelete(ctx, bson.M{"_id": stableID(ticket), "actor_id": actor, "expires_at": bson.M{"$gt": time.Now().UTC()}}).Decode(&grant)
	if err != nil {
		return PublicSettings{}, err
	}
	stage = "create_temporary_file"
	file, err := os.CreateTemp("", "orix-apk-*")
	if err != nil {
		return PublicSettings{}, err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	stage = "receive_and_verify_upload"
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(source, maxAPKSize+1))
	if err != nil || size != grant.Size || size > maxAPKSize || hex.EncodeToString(hash.Sum(nil)) != grant.Digest {
		return PublicSettings{}, errInvalidAPK
	}
	stage = "validate_apk_archive"
	archive, err := zip.NewReader(file, size)
	if err != nil {
		return PublicSettings{}, errInvalidAPK
	}
	manifest, dex := false, false
	for _, entry := range archive.File {
		if entry.Name == "AndroidManifest.xml" {
			manifest = true
		}
		if entry.Name == "classes.dex" {
			dex = true
		}
	}
	if !manifest || !dex {
		return PublicSettings{}, errInvalidAPK
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return PublicSettings{}, err
	}
	stage = "store_apk"
	bucket := m.collection.Database().GridFSBucket(options.GridFSBucket().SetName("android_releases"))
	id, err := bucket.UploadFromStream(ctx, "Orix.apk", file)
	if err != nil {
		return PublicSettings{}, err
	}
	stage = "start_publication_transaction"
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		_ = bucket.Delete(context.WithoutCancel(ctx), id)
		return PublicSettings{}, err
	}
	defer session.EndSession(ctx)
	stage = "commit_publication"
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		settings, err := m.PublicSettings(tx)
		if err != nil {
			return nil, err
		}
		old := settings.APKID
		_, err = m.collection.Database().Collection("app_config").UpdateOne(tx, bson.M{"_id": "public"}, bson.M{"$set": bson.M{"apk_id": id, "apk_version": grant.Version, "apk_sha256": grant.Digest, "apk_size": size}}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return nil, err
		}
		if old != nil {
			_, err = m.collection.Database().Collection("apk_cleanup").UpdateOne(tx, bson.M{"_id": *old}, bson.M{"$set": bson.M{"created_at": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true))
			if err != nil {
				return nil, err
			}
		}
		_, err = WriteAudit(tx, m.collection.Database(), bson.M{"actor_id": actor, "action": "published_android_release", "version": grant.Version, "sha256": grant.Digest, "created_at": time.Now().UTC()})
		return nil, err
	})
	if err != nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		_ = bucket.Delete(cleanup, id)
		return PublicSettings{}, err
	}
	stage = "read_published_settings"
	m.cleanupAPK(ctx)
	return m.PublicSettings(ctx)
}
func (m mongoUsers) OpenAPK(ctx context.Context) (*mongo.GridFSDownloadStream, PublicSettings, error) {
	settings, err := m.PublicSettings(ctx)
	if err != nil {
		return nil, settings, err
	}
	if settings.APKID == nil {
		return nil, settings, mongo.ErrNoDocuments
	}
	stream, err := m.collection.Database().GridFSBucket(options.GridFSBucket().SetName("android_releases")).OpenDownloadStream(ctx, *settings.APKID)
	return stream, settings, err
}
func (m mongoUsers) cleanupAPK(ctx context.Context) {
	var job struct {
		ID bson.ObjectID `bson:"_id"`
	}
	db := m.collection.Database()
	if db.Collection("apk_cleanup").FindOne(ctx, bson.M{}).Decode(&job) != nil {
		return
	}
	current, err := m.PublicSettings(ctx)
	if err != nil || current.APKID != nil && *current.APKID == job.ID {
		return
	}
	// Delete both collections idempotently, including a retry after partial failure.
	if _, err = db.Collection("android_releases.chunks").DeleteMany(ctx, bson.M{"files_id": job.ID}); err != nil {
		return
	}
	if _, err = db.Collection("android_releases.files").DeleteOne(ctx, bson.M{"_id": job.ID}); err != nil {
		return
	}
	_, _ = db.Collection("apk_cleanup").DeleteOne(ctx, bson.M{"_id": job.ID})
}
func (m mongoUsers) RunDownloadCleanup(ctx context.Context) {
	timer := time.NewTicker(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			work, cancel := context.WithTimeout(ctx, 20*time.Second)
			m.cleanupAPK(work)
			_, _ = m.collection.Database().Collection("apk_upload_tickets").DeleteMany(work, bson.M{"expires_at": bson.M{"$lte": time.Now().UTC()}})
			cancel()
		}
	}
}
func (a *AuthDependencies) downloadRepository() (DownloadRepository, bool) {
	if a == nil {
		return nil, false
	}
	repo, ok := a.Users.(DownloadRepository)
	return repo, ok
}
func (a *AuthDependencies) publicSettings(w http.ResponseWriter, r *http.Request) {
	repo, ok := a.downloadRepository()
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Download settings unavailable"})
		return
	}
	value, err := repo.PublicSettings(r.Context())
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Download settings unavailable"})
		return
	}
	writeJSON(w, 200, value)
}
func (a *AuthDependencies) setPublicSettings(w http.ResponseWriter, r *http.Request, user User) {
	var value PublicSettings
	if decodeBody(r, &value) != nil || len(value.SupportEmail) > 254 || !strings.Contains(value.SupportEmail, "@") {
		writeJSON(w, 422, map[string]string{"detail": "Provide a support email and valid settings"})
		return
	}
	if value.PlayStoreURL != "" {
		link, err := url.Parse(value.PlayStoreURL)
		if err != nil || link.Scheme != "https" || link.Host != "play.google.com" || link.User != nil || link.Query().Get("id") != "com.orixridesharestudents.app" {
			writeJSON(w, 422, map[string]string{"detail": "Use the official Orix Play Store link"})
			return
		}
	}
	repo, ok := a.downloadRepository()
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Settings unavailable"})
		return
	}
	if err := repo.SetPublicSettings(r.Context(), user.ID, value); err != nil {
		if adminAccessDenied(w, err) {
			return
		}
		writeJSON(w, 503, map[string]string{"detail": "Settings could not be saved"})
		return
	}
	a.publicSettings(w, r)
}
func (a *AuthDependencies) apkTicket(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Version string `json:"version"`
		Digest  string `json:"sha256"`
		Size    int64  `json:"size"`
	}
	if decodeBody(r, &input) != nil {
		writeJSON(w, 422, map[string]string{"detail": "Provide release metadata"})
		return
	}
	_, versionErr := versionParts(input.Version)
	digest, err := hex.DecodeString(input.Digest)
	if versionErr != nil || err != nil || len(digest) != 32 || input.Size < 1 || input.Size > maxAPKSize {
		writeJSON(w, 422, map[string]string{"detail": "Provide a stable version, SHA-256 and size up to 256 MB"})
		return
	}
	repo, ok := a.downloadRepository()
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Release storage unavailable"})
		return
	}
	ticket, err := repo.CreateAPKTicket(r.Context(), user.ID, input.Version, strings.ToLower(input.Digest), input.Size)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Could not authorize upload"})
		return
	}
	writeJSON(w, 200, map[string]string{"upload_ticket": ticket})
}
func (a *AuthDependencies) uploadAPK(w http.ResponseWriter, r *http.Request, user User) {
	if user.Role != "admin" {
		writeJSON(w, 403, map[string]string{"detail": "Administrator access required"})
		return
	}
	ticket := r.Header.Get("X-Upload-Ticket")
	if len(ticket) != 64 {
		writeJSON(w, 403, map[string]string{"detail": "A fresh upload grant is required"})
		return
	}
	repo, ok := a.downloadRepository()
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Release storage unavailable"})
		return
	}
	control := http.NewResponseController(w)
	_ = control.SetReadDeadline(time.Now().Add(5 * time.Minute))
	_ = control.SetWriteDeadline(time.Now().Add(5 * time.Minute))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	value, err := repo.PublishAPK(ctx, user.ID, ticket, r.Body)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, errInvalidAPK) {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, map[string]string{"detail": "Upload was not published. Please try again."})
		return
	}
	writeJSON(w, 200, value)
}
func (a *AuthDependencies) downloadAPK(w http.ResponseWriter, r *http.Request) {
	repo, ok := a.downloadRepository()
	if !ok {
		http.NotFound(w, r)
		return
	}
	stream, value, err := repo.OpenAPK(r.Context())
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer stream.Close()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Minute))
	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	w.Header().Set("Content-Disposition", `attachment; filename="Orix.apk"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("ETag", `"`+value.APKDigest+`"`)
	_, _ = io.Copy(w, stream)
}
