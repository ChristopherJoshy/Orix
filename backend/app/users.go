package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ProfileRepository interface {
	UpdateProfile(context.Context, string, map[string]any) (User, error)
	CompleteOnboarding(context.Context, string, string, *string) error
}

func (m mongoUsers) UpdateProfile(ctx context.Context, id string, changes map[string]any) (User, error) {
	changes["updated_at"] = time.Now().UTC()
	token, hasToken := changes["fcm_token"].(string)
	if !hasToken || token == "" {
		var user User
		err := m.collection.FindOneAndUpdate(ctx, bson.M{"user_id": id}, bson.M{"$set": changes}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&user)
		return user, err
	}
	// A stable claim document serializes concurrent ownership changes for the same
	// installation token, including when legacy profiles still contain duplicates.
	hash := sha256.Sum256([]byte(token))
	claim := hex.EncodeToString(hash[:])
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return User{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		claims := m.collection.Database().Collection("push_token_claims")
		// A profile currently stores one token. Retire its previous claim when
		// this account rotates tokens, without disturbing another owner's claim.
		var previous struct {
			Token string `bson:"fcm_token"`
		}
		if err := m.collection.FindOne(tx, bson.M{"user_id": id}, options.FindOne().SetProjection(bson.M{"fcm_token": 1})).Decode(&previous); err != nil {
			return nil, err
		}
		if previous.Token != "" && previous.Token != token {
			oldHash := sha256.Sum256([]byte(previous.Token))
			if _, err := claims.DeleteOne(tx, bson.M{"_id": hex.EncodeToString(oldHash[:]), "user_id": id}); err != nil {
				return nil, err
			}
		}
		if _, err := claims.UpdateOne(tx, bson.M{"_id": claim}, bson.M{"$set": bson.M{"user_id": id, "updated_at": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true)); err != nil {
			return nil, err
		}
		if _, err := m.collection.UpdateMany(tx, bson.M{"fcm_token": token, "user_id": bson.M{"$ne": id}}, bson.M{"$unset": bson.M{"fcm_token": ""}}); err != nil {
			return nil, err
		}
		var user User
		err := m.collection.FindOneAndUpdate(tx, bson.M{"user_id": id}, bson.M{"$set": changes}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&user)
		return user, err
	})
	if err != nil {
		return User{}, err
	}
	return result.(User), nil
}

// releasePushToken compares both owner and token so an older device cannot
// unregister a token already reassigned to another account or device.
func (m mongoUsers) releasePushToken(ctx context.Context, user, token string) error {
	hash := sha256.Sum256([]byte(token))
	claim := hex.EncodeToString(hash[:])
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.collection.UpdateOne(tx, bson.M{"user_id": user, "fcm_token": token}, bson.M{"$unset": bson.M{"fcm_token": ""}}); err != nil {
			return nil, err
		}
		_, err := m.collection.Database().Collection("push_token_claims").DeleteOne(tx, bson.M{"_id": claim, "user_id": user})
		return nil, err
	})
	return err
}
func (a *AuthDependencies) unregisterPushToken(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Token string `json:"token"`
	}
	if decodeBody(r, &input) != nil || input.Token == "" || len(input.Token) > 4096 {
		writeJSON(w, 422, map[string]string{"detail": "Invalid push token"})
		return
	}
	repository, ok := a.Users.(interface {
		releasePushToken(context.Context, string, string) error
	})
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Push token storage unavailable"})
		return
	}
	if err := repository.releasePushToken(r.Context(), user.ID, input.Token); err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to unregister push token"})
		return
	}
	writeJSON(w, 200, map[string]bool{"success": true})
}

func (m mongoUsers) CompleteOnboarding(ctx context.Context, id, phone string, gender *string) error {
	// First completion wins; retrying the request cannot replace consent/profile data.
	now := time.Now().UTC()
	_, err := m.collection.UpdateOne(ctx, bson.M{"user_id": id, "onboarding_completed": bson.M{"$ne": true}}, bson.M{"$set": bson.M{
		"phone": phone, "gender": gender, "safety_consent": true, "privacy_consent": true, "privacy_consent_at": now, "onboarding_completed": true,
		"consent_at": now, "updated_at": now,
	}})
	return err
}

func accountRestricted(user User) bool {
	return user.Status == "deleting" || user.Status == "deleted" || user.Status == "banned" || (user.Status == "suspended" && (user.SuspensionExpires == nil || user.SuspensionExpires.After(time.Now())))
}
func (a *AuthDependencies) authenticated(next func(http.ResponseWriter, *http.Request, User)) http.HandlerFunc {
	return a.identified(next, false)
}
func (a *AuthDependencies) supportAuthenticated(next func(http.ResponseWriter, *http.Request, User)) http.HandlerFunc {
	return a.identified(next, true)
}
func (a *AuthDependencies) identified(next func(http.ResponseWriter, *http.Request, User), allowAppeals bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a == nil || a.Verifier == nil || a.Users == nil {
			writeJSON(w, 503, map[string]string{"detail": "Authentication is not configured"})
			return
		}
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) > 16384 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSON(w, 401, map[string]string{"detail": "Bearer token required"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		identity, err := a.Verifier.Verify(ctx, parts[1])
		if err != nil {
			writeJSON(w, 401, map[string]string{"detail": "Invalid or expired token"})
			return
		}
		user, _, err := a.Users.GetOrCreate(ctx, identity)
		if err != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to load your account"})
			return
		}
		appealAccess := allowAppeals && (user.Status == "banned" || user.Status == "suspended")
		if accountRestricted(user) && !appealAccess {
			writeJSON(w, 403, map[string]any{"detail": map[string]string{"error": "account_" + user.Status, "message": "Your account is " + user.Status + "."}})
			return
		}
		scoped := r.WithContext(ctx)
		if !a.maintenanceGate(w, scoped, user) {
			return
		}
		next(w, scoped, user)
	}
}
func decodeBody(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}

type profileResponse struct {
	Role string `json:"role"`
	User
	Phone     *string        `json:"phone"`
	Gender    *string        `json:"gender"`
	IsRider   bool           `json:"is_rider"`
	RiderInfo map[string]any `json:"rider_info"`
	Status    string         `json:"status"`
}

func profile(user User) profileResponse {
	return profileResponse{Role: user.Role, User: user, Phone: user.Phone, Gender: user.Gender, IsRider: user.IsRider, RiderInfo: user.RiderInfo, Status: user.Status}
}
func (a *AuthDependencies) getProfile(w http.ResponseWriter, r *http.Request, user User) {
	writeJSON(w, 200, profile(user))
}
func (a *AuthDependencies) updateProfile(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Name       *string `json:"display_name"`
		Photo      *string `json:"photo_url"`
		Token      *string `json:"fcm_token"`
		Occupation *string `json:"occupation"`
	}
	if decodeBody(r, &input) != nil {
		writeJSON(w, 422, map[string]string{"detail": "Invalid profile fields"})
		return
	}
	changes := map[string]any{}
	if input.Occupation != nil {
		if !validOccupation(*input.Occupation) {
			writeJSON(w, 422, map[string]string{"detail": "Choose student, teacher or other"})
			return
		}
		changes["occupation"] = *input.Occupation
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || utf8.RuneCountInString(name) > 100 {
			writeJSON(w, 422, map[string]string{"detail": "Display name must contain 1–100 characters"})
			return
		}
		changes["display_name"] = name
	}
	if input.Photo != nil {
		parsed, err := url.Parse(*input.Photo)
		local := err == nil && parsed.Scheme == "" && parsed.Host == "" && parsed.Path == "/api/v1/users/"+url.PathEscape(user.ID)+"/avatar"
		if len(*input.Photo) > 2048 || (*input.Photo != "" && !local && (err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil)) {
			writeJSON(w, 422, map[string]string{"detail": "Photo must use HTTPS or an uploaded image path"})
			return
		}
		changes["photo_url"] = *input.Photo
	}
	if input.Token != nil {
		if len(*input.Token) > 4096 {
			writeJSON(w, 422, map[string]string{"detail": "Invalid push token"})
			return
		}
		changes["fcm_token"] = *input.Token
	}
	if len(changes) == 0 {
		writeJSON(w, 200, profile(user))
		return
	}
	repository, ok := a.Users.(ProfileRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Profile storage is not configured"})
		return
	}
	updated, err := repository.UpdateProfile(r.Context(), user.ID, changes)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to update your profile"})
		return
	}
	writeJSON(w, 200, profile(updated))
}

type IntroductionPreferences struct {
	Occupation  string `json:"occupation" bson:"occupation"`
	OfferLifts  bool   `json:"offer_lifts" bson:"offer_lifts"`
	PhotoClear  bool   `json:"photo_clear" bson:"photo_clear"`
	TourVersion int    `json:"tour_version" bson:"tour_version"`
}
type IntroductionRepository interface {
	CompleteIntroduction(context.Context, string, string, *string, IntroductionPreferences) error
}

func (m mongoUsers) CompleteIntroduction(ctx context.Context, id, phone string, gender *string, prefs IntroductionPreferences) error {
	now := time.Now().UTC()
	_, err := m.collection.UpdateOne(ctx, bson.M{"user_id": id, "onboarding_completed": bson.M{"$ne": true}}, bson.M{"$set": bson.M{
		"phone": phone, "gender": gender, "safety_consent": true, "privacy_consent": true,
		"privacy_consent_at": now, "consent_at": now, "onboarding_completed": true,
		"introduction_preferences": prefs, "occupation": prefs.Occupation, "updated_at": now,
	}})
	return err
}

var phonePattern = regexp.MustCompile(`^\+?[0-9]{10,15}$`)

func (a *AuthDependencies) onboard(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Phone          string  `json:"phone"`
		Gender         *string `json:"gender"`
		Consent        bool    `json:"safety_consent"`
		PrivacyConsent bool    `json:"privacy_consent"`
		IntroductionPreferences
	}
	if decodeBody(r, &input) != nil || !phonePattern.MatchString(input.Phone) || (input.Gender != nil && *input.Gender != "M" && *input.Gender != "F" && *input.Gender != "O") {
		writeJSON(w, 422, map[string]string{"detail": "Provide a valid phone number and gender"})
		return
	}
	if input.Occupation == "" {
		input.Occupation = "other"
	}
	if !validOccupation(input.Occupation) {
		writeJSON(w, 422, map[string]string{"detail": "Choose student, teacher or other"})
		return
	}
	if !input.PrivacyConsent {
		writeJSON(w, 400, map[string]string{"detail": "You must acknowledge the privacy policy to continue"})
		return
	}
	if !input.Consent {
		writeJSON(w, 400, map[string]string{"detail": "You must agree to the safety terms to continue"})
		return
	}
	if input.TourVersion < 0 || input.TourVersion > 1 ||
		(input.TourVersion == 1 && (!input.PhotoClear || user.Photo == nil || *user.Photo == "")) {
		writeJSON(w, 422, map[string]string{"detail": "Add a clear profile photo and confirm it shows your face"})
		return
	}
	if !user.Onboarded && input.TourVersion == 1 {
		repository, ok := a.Users.(IntroductionRepository)
		if !ok || repository.CompleteIntroduction(r.Context(), user.ID, input.Phone, input.Gender, input.IntroductionPreferences) != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to save your introduction"})
			return
		}
	} else if !user.Onboarded {
		repository, ok := a.Users.(ProfileRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Profile storage is not configured"})
			return
		}
		if repository.CompleteOnboarding(r.Context(), user.ID, input.Phone, input.Gender) != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to complete onboarding"})
			return
		}
	}
	writeJSON(w, 200, map[string]any{"success": true, "message": "Onboarding completed successfully"})
}

func validOccupation(value string) bool {
	return value == "student" || value == "teacher" || value == "other"
}
