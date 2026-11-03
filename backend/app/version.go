package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type AppVersion struct {
	Current  string  `json:"current_version" bson:"current_version"`
	Minimum  string  `json:"min_version" bson:"min_version"`
	Required bool    `json:"update_required" bson:"-"`
	Title    *string `json:"update_title" bson:"update_title"`
	Notes    *string `json:"update_notes" bson:"update_notes"`
	PlayURL  *string `json:"play_store_url" bson:"play_store_url"`
	AppleURL *string `json:"app_store_url" bson:"app_store_url"`
}
type VersionRepository interface {
	AppVersion(context.Context) (AppVersion, error)
	SetAppVersion(context.Context, string, AppVersion) error
}

func defaultVersion() AppVersion { return AppVersion{Current: Version, Minimum: "2.0.0"} }
func (m mongoUsers) AppVersion(ctx context.Context) (AppVersion, error) {
	version := defaultVersion()
	err := m.collection.Database().Collection("app_config").FindOne(ctx, bson.M{"_id": "version"}).Decode(&version)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return defaultVersion(), nil
	}
	return version, err
}
func (m mongoUsers) SetAppVersion(ctx context.Context, admin string, version AppVersion) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.authorizeAdminMutation(tx, admin); err != nil {
			return nil, err
		}
		db := m.collection.Database()
		if _, err := db.Collection("app_config").UpdateOne(tx, bson.M{"_id": "version"}, bson.M{"$set": version}, options.UpdateOne().SetUpsert(true)); err != nil {
			return nil, err
		}
		_, err := WriteAudit(tx, db, bson.M{"actor_id": admin, "action": "updated_app_version", "current_version": version.Current, "min_version": version.Minimum, "created_at": time.Now().UTC()})
		return nil, err
	})
	return err
}

var stableVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\+[0-9]+)?$`)

func versionParts(value string) ([3]uint64, error) {
	match := stableVersionPattern.FindStringSubmatch(value)
	parts := [3]uint64{}
	if match == nil || len(value) > 64 {
		return parts, errors.New("invalid version")
	}
	for i := range parts {
		parsed, err := strconv.ParseUint(match[i+1], 10, 32)
		if err != nil {
			return parts, err
		}
		parts[i] = parsed
	}
	return parts, nil
}
func compareVersions(a, b string) (int, error) {
	first, err := versionParts(a)
	if err != nil {
		return 0, err
	}
	second, err := versionParts(b)
	if err != nil {
		return 0, err
	}
	for i := range first {
		if first[i] < second[i] {
			return -1, nil
		}
		if first[i] > second[i] {
			return 1, nil
		}
	}
	return 0, nil
}
func validVersionConfig(version AppVersion) bool {
	order, err := compareVersions(version.Minimum, version.Current)
	if err != nil || order > 0 {
		return false
	}
	if version.Title != nil && len(*version.Title) > 200 || version.Notes != nil && len(*version.Notes) > 10000 {
		return false
	}
	for _, link := range []struct {
		value *string
		host  string
	}{{version.PlayURL, "play.google.com"}, {version.AppleURL, "apps.apple.com"}} {
		if link.value == nil {
			continue
		}
		parsed, err := url.Parse(*link.value)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() != link.host || parsed.User != nil || parsed.Port() != "" {
			return false
		}
	}
	return true
}
func (a *AuthDependencies) appVersion(w http.ResponseWriter, r *http.Request) {
	version := defaultVersion()
	if a != nil {
		if repository, ok := a.Users.(VersionRepository); ok {
			stored, err := repository.AppVersion(r.Context())
			if err != nil {
				writeJSON(w, 503, map[string]string{"detail": "Version information unavailable"})
				return
			}
			version = stored
		}
	}
	if !validVersionConfig(version) {
		writeJSON(w, 503, map[string]string{"detail": "Version configuration needs administrator review"})
		return
	}
	if client := r.URL.Query().Get("client_version"); client != "" {
		comparison, err := compareVersions(client, version.Minimum)
		if err != nil {
			writeJSON(w, 422, map[string]string{"detail": "Use major.minor.patch with optional build number"})
			return
		}
		version.Required = comparison < 0
	}
	writeJSON(w, 200, version)
}
func (a *AuthDependencies) setAppVersion(w http.ResponseWriter, r *http.Request, user User) {
	var input AppVersion
	if decodeBody(r, &input) != nil || !validVersionConfig(input) {
		writeJSON(w, 422, map[string]string{"detail": "Provide valid stable versions and official HTTPS store links"})
		return
	}
	repository, ok := a.Users.(VersionRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Version storage unavailable"})
		return
	}
	input.Required = false
	if err := repository.SetAppVersion(r.Context(), user.ID, input); err != nil {
		if adminAccessDenied(w, err) {
			return
		}
		writeJSON(w, 503, map[string]string{"detail": "Unable to save version configuration"})
		return
	}
	writeJSON(w, 200, input)
}
