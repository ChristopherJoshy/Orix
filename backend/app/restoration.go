package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type RestorationRepository interface {
	RestoreAccount(context.Context, string, string, string) error
}

func (m mongoUsers) RestoreAccount(ctx context.Context, actor, target, note string) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		if actor == target {
			return nil, ErrModerationConflict
		}
		var user User
		if err := m.collection.FindOne(tx, bson.M{"user_id": target}).Decode(&user); err != nil {
			return nil, err
		}
		if user.Role == "admin" || user.Status == "deleting" || user.Status == "deleted" {
			return nil, ErrModerationConflict
		}
		if user.Status == "active" {
			return nil, nil
		}
		if user.Status != "banned" && user.Status != "suspended" {
			return nil, ErrModerationConflict
		}
		now := time.Now().UTC()
		db := m.collection.Database()
		if _, err := m.collection.UpdateOne(tx, bson.M{"user_id": target, "status": user.Status}, bson.M{"$set": bson.M{"status": "active", "updated_at": now}, "$unset": bson.M{"ban_reason": "", "suspension_expires_at": ""}}); err != nil {
			return nil, err
		}
		if _, err := WriteAudit(tx, db, bson.M{"actor_id": actor, "target_id": target, "action": "account_restored", "before_status": user.Status, "note": note, "created_at": now}); err != nil {
			return nil, err
		}
		_, err := db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "account_restored", "target_user_id": target, "status": "pending", "attempts": 0, "created_at": now})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}
func (a *AuthDependencies) restoreAccount(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Note string `json:"note"`
	}
	if decodeBody(r, &input) != nil || strings.TrimSpace(input.Note) == "" || utf8.RuneCountInString(input.Note) > 1000 || r.PathValue("user_id") == "" || len(r.PathValue("user_id")) > 100 {
		writeJSON(w, 422, map[string]string{"detail": "Provide a target account and a review note"})
		return
	}
	repository, ok := a.Users.(RestorationRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Account restoration unavailable"})
		return
	}
	err := repository.RestoreAccount(r.Context(), user.ID, r.PathValue("user_id"), strings.TrimSpace(input.Note))
	if adminAccessDenied(w, err) {
		return
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Account not found"})
		return
	}
	if errors.Is(err, ErrModerationConflict) {
		writeJSON(w, 409, map[string]string{"detail": "This account needs a separate administrator recovery or cannot be restored"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to restore account"})
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Account access restored. Cancelled trips remain cancelled."})
}
