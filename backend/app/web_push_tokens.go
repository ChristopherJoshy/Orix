package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type WebPushTokenRepository interface {
	RegisterWebPush(context.Context, string, string) error
	UnregisterWebPush(context.Context, string, string) error
}

func tokenDigest(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
func (m mongoUsers) RegisterWebPush(ctx context.Context, user, token string) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		var current User
		if err := m.collection.FindOne(tx, bson.M{"user_id": user}).Decode(&current); err != nil {
			return nil, err
		}
		if accountRestricted(current) || !current.Onboarded {
			return nil, ErrGroupUnavailable
		}
		if _, err := m.collection.UpdateOne(tx, bson.M{"user_id": user}, bson.M{"$inc": bson.M{"push_registration_version": 1}}); err != nil {
			return nil, err
		}
		tokens := m.collection.Database().Collection("web_push_tokens")
		id := tokenDigest(token)
		if _, err := tokens.DeleteMany(tx, bson.M{"user_id": user, "updated_at": bson.M{"$lt": time.Now().UTC().Add(-30 * 24 * time.Hour)}}); err != nil {
			return nil, err
		}
		count, err := tokens.CountDocuments(tx, bson.M{"user_id": user, "_id": bson.M{"$ne": id}})
		if err != nil {
			return nil, err
		}
		if count >= 10 {
			return nil, ErrTicketLimit
		}
		_, err = tokens.UpdateOne(tx, bson.M{"_id": id}, bson.M{"$set": bson.M{"user_id": user, "token": token, "updated_at": time.Now().UTC()}}, options.UpdateOne().SetUpsert(true))
		return nil, err
	})
	return err
}
func (m mongoUsers) UnregisterWebPush(ctx context.Context, user, token string) error {
	_, err := m.collection.Database().Collection("web_push_tokens").DeleteOne(ctx, bson.M{"_id": tokenDigest(token), "user_id": user})
	return err
}
func (a *AuthDependencies) webPushToken(w http.ResponseWriter, r *http.Request, user User) {
	if !user.Onboarded {
		writeJSON(w, 403, map[string]string{"detail": "Complete onboarding first"})
		return
	}
	var input struct {
		Token string `json:"token"`
	}
	if decodeBody(r, &input) != nil || len(input.Token) < 16 || len(input.Token) > 4096 {
		writeJSON(w, 422, map[string]string{"detail": "Invalid token"})
		return
	}
	repository, ok := a.Users.(WebPushTokenRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Notifications unavailable"})
		return
	}
	var err error
	if r.Method == http.MethodPost {
		err = repository.RegisterWebPush(r.Context(), user.ID, input.Token)
	} else {
		err = repository.UnregisterWebPush(r.Context(), user.ID, input.Token)
	}
	if errors.Is(err, ErrTicketLimit) {
		writeJSON(w, 409, map[string]string{"detail": "Device limit reached"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to update notifications"})
		return
	}
	writeJSON(w, 200, map[string]bool{"success": true})
}
