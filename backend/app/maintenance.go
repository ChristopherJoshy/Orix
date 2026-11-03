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
)

type MaintenanceState struct {
	Enabled bool   `json:"maintenance_mode" bson:"enabled"`
	Message string `json:"message" bson:"message"`
}
type MaintenanceRepository interface {
	Maintenance(context.Context) (MaintenanceState, error)
	SetMaintenance(context.Context, string, MaintenanceState) error
}

func (m mongoUsers) Maintenance(ctx context.Context) (MaintenanceState, error) {
	var state MaintenanceState
	err := m.collection.Database().Collection("app_config").FindOne(ctx, bson.M{"_id": "maintenance"}).Decode(&state)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return state, nil
	}
	return state, err
}
func (m mongoUsers) SetMaintenance(ctx context.Context, admin string, state MaintenanceState) error {
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
		if _, err := db.Collection("app_config").UpdateOne(tx, bson.M{"_id": "maintenance"}, bson.M{"$set": state}, options.UpdateOne().SetUpsert(true)); err != nil {
			return nil, err
		}
		_, err := WriteAudit(tx, db, bson.M{"actor_id": admin, "action": "maintenance_changed", "enabled": state.Enabled, "created_at": time.Now().UTC()})
		return nil, err
	})
	return err
}
func maintenanceSafeRequest(r *http.Request) bool {
	if r.Method == http.MethodGet {
		return true
	}
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/reports") || strings.HasPrefix(path, "/api/v1/support/") || strings.HasSuffix(path, "/block") || strings.HasPrefix(path, "/api/v1/notifications") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/groups/") {
		return true
	}
	return path == "/api/v1/users/me/accept-terms" || path == "/api/v1/users/me/notification-preferences" || path == "/api/v1/users/me" && r.Method == http.MethodDelete
}
func (a *AuthDependencies) maintenanceGate(w http.ResponseWriter, r *http.Request, user User) bool {
	if user.Role == "admin" || maintenanceSafeRequest(r) {
		return true
	}
	repository, ok := a.Users.(MaintenanceRepository)
	if !ok {
		return true
	}
	state, err := repository.Maintenance(r.Context())
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Service status unavailable"})
		return false
	}
	if !state.Enabled {
		return true
	}
	message := state.Message
	if message == "" {
		message = "New ride activity is temporarily paused. Existing trip and safety controls remain available."
	}
	writeJSON(w, 503, map[string]any{"detail": map[string]string{"error": "maintenance_mode", "message": message}})
	return false
}
func (a *AuthDependencies) serviceStatus(w http.ResponseWriter, r *http.Request) {
	state := MaintenanceState{}
	if a != nil {
		if repository, ok := a.Users.(MaintenanceRepository); ok {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			stored, err := repository.Maintenance(ctx)
			if err != nil {
				writeJSON(w, 503, map[string]string{"detail": "Service status unavailable"})
				return
			}
			state = stored
		}
	}
	writeJSON(w, 200, state)
}
func (a *AuthDependencies) changeMaintenance(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Enabled *bool  `json:"enabled"`
		Message string `json:"message"`
	}
	if decodeBody(r, &input) != nil || input.Enabled == nil || utf8.RuneCountInString(input.Message) > 400 {
		writeJSON(w, 422, map[string]string{"detail": "Provide an enabled flag and a short message"})
		return
	}
	repository, ok := a.Users.(MaintenanceRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Maintenance storage unavailable"})
		return
	}
	state := MaintenanceState{*input.Enabled, strings.TrimSpace(input.Message)}
	if err := repository.SetMaintenance(r.Context(), user.ID, state); err != nil {
		if adminAccessDenied(w, err) {
			return
		}
		writeJSON(w, 503, map[string]string{"detail": "Unable to change service status"})
		return
	}
	writeJSON(w, 200, state)
}
