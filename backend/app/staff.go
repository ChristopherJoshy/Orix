package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

var ErrAdminAccess = errors.New("administrator authorization is no longer available")

type StaffRepository interface {
	ChangeRole(context.Context, string, string, string) error
}

func (m mongoUsers) ChangeRole(ctx context.Context, actor, target, role string) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		// Serialize all staff changes before counting the remaining administrators.
		if _, err := db.Collection("app_config").UpdateOne(tx, bson.M{"_id": "staff_role_lock"}, bson.M{"$inc": bson.M{"version": 1}}, options.UpdateOne().SetUpsert(true)); err != nil {
			return nil, err
		}
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		var user User
		if err := m.collection.FindOne(tx, bson.M{"user_id": target}).Decode(&user); err != nil {
			return nil, err
		}
		if user.Role == role {
			return nil, nil
		}
		if role == "admin" && user.Status != "active" {
			return nil, ErrModerationConflict
		}
		if user.Role == "admin" && role == "user" {
			count, err := m.collection.CountDocuments(tx, bson.M{"role": "admin", "status": "active"})
			if err != nil {
				return nil, err
			}
			if count <= 1 {
				return nil, ErrModerationConflict
			}
		}
		now := time.Now().UTC()
		if _, err := m.collection.UpdateOne(tx, bson.M{"user_id": target}, bson.M{"$set": bson.M{"role": role, "updated_at": now}}); err != nil {
			return nil, err
		}
		_, err := WriteAudit(tx, db, bson.M{"actor_id": actor, "action": "staff_role_changed", "target_id": target, "before_role": user.Role, "after_role": role, "created_at": now})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}
func (a *AuthDependencies) changeRole(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Role string `json:"role"`
	}
	if decodeBody(r, &input) != nil || input.Role != "admin" && input.Role != "user" || len(r.PathValue("user_id")) > 100 {
		writeJSON(w, 422, map[string]string{"detail": "Choose user or admin role"})
		return
	}
	repository, ok := a.Users.(StaffRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Staff storage unavailable"})
		return
	}
	err := repository.ChangeRole(r.Context(), user.ID, r.PathValue("user_id"), input.Role)
	if adminAccessDenied(w, err) {
		return
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Account not found"})
		return
	}
	if errors.Is(err, ErrModerationConflict) {
		writeJSON(w, 409, map[string]string{"detail": "Role change would remove the last active administrator or affect a restricted account"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to update staff role"})
		return
	}
	writeJSON(w, 200, map[string]bool{"success": true})
}

// authorizeAdminMutation must run in the mutation's Mongo transaction. Writing
// the actor document makes concurrent demotion/deletion conflict and retry; a
// read-only role check in a snapshot could otherwise commit after access changes.
func (m mongoUsers) authorizeAdminMutation(ctx context.Context, id string) (User, error) {
	var actor User
	err := m.collection.FindOne(ctx, bson.M{"user_id": id, "role": "admin"}).Decode(&actor)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return actor, ErrAdminAccess
	}
	if err != nil {
		return actor, err
	}
	if accountRestricted(actor) {
		return actor, ErrAdminAccess
	}
	result, err := m.collection.UpdateOne(ctx, bson.M{"user_id": id, "role": "admin", "status": actor.Status}, bson.M{"$inc": bson.M{"authorization_version": 1}})
	if err != nil {
		return actor, err
	}
	if result.MatchedCount != 1 {
		return actor, ErrAdminAccess
	}
	return actor, nil
}

func adminAccessDenied(w http.ResponseWriter, err error) bool {
	if !errors.Is(err, ErrAdminAccess) {
		return false
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"detail": "Administrator access is no longer available. Sign in again or contact an administrator."})
	return true
}
