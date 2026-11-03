package server

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type NotificationPreferences struct {
	RideReminders bool `json:"ride_reminders" bson:"ride_reminders"`
	ChatMessages  bool `json:"chat_messages" bson:"chat_messages"`
	GroupUpdates  bool `json:"group_updates" bson:"group_updates"`
	Promotions    bool `json:"promotions" bson:"promotions"`
}
type PreferenceRepository interface {
	GetPreferences(context.Context, string) (NotificationPreferences, error)
	SetPreferences(context.Context, string, map[string]bool) (NotificationPreferences, error)
}

func (m mongoUsers) GetPreferences(ctx context.Context, user string) (NotificationPreferences, error) {
	var record struct {
		Preferences map[string]bool `bson:"notification_preferences"`
	}
	if err := m.collection.FindOne(ctx, bson.M{"user_id": user}).Decode(&record); err != nil {
		return NotificationPreferences{}, err
	}
	defaults := map[string]bool{"ride_reminders": true, "chat_messages": true, "group_updates": true, "promotions": false}
	for key, value := range record.Preferences {
		if _, ok := defaults[key]; ok {
			defaults[key] = value
		}
	}
	return NotificationPreferences{defaults["ride_reminders"], defaults["chat_messages"], defaults["group_updates"], defaults["promotions"]}, nil
}
func (m mongoUsers) SetPreferences(ctx context.Context, user string, changes map[string]bool) (NotificationPreferences, error) {
	fields := bson.M{"updated_at": time.Now().UTC()}
	for key, value := range changes {
		fields["notification_preferences."+key] = value
	}
	if _, err := m.collection.UpdateOne(ctx, bson.M{"user_id": user}, bson.M{"$set": fields}); err != nil {
		return NotificationPreferences{}, err
	}
	return m.GetPreferences(ctx, user)
}
func (a *AuthDependencies) preferences(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.Users.(PreferenceRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Preference storage unavailable"})
		return
	}
	if r.Method == http.MethodGet {
		result, err := repository.GetPreferences(r.Context(), user.ID)
		if err != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to load preferences"})
			return
		}
		writeJSON(w, 200, result)
		return
	}
	var input struct {
		Ride       *bool `json:"ride_reminders"`
		Chat       *bool `json:"chat_messages"`
		Groups     *bool `json:"group_updates"`
		Promotions *bool `json:"promotions"`
	}
	if decodeBody(r, &input) != nil {
		writeJSON(w, 422, map[string]string{"detail": "Invalid preference fields"})
		return
	}
	changes := map[string]bool{}
	for key, value := range map[string]*bool{"ride_reminders": input.Ride, "chat_messages": input.Chat, "group_updates": input.Groups, "promotions": input.Promotions} {
		if value != nil {
			changes[key] = *value
		}
	}
	result, err := repository.SetPreferences(r.Context(), user.ID, changes)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to save preferences"})
		return
	}
	writeJSON(w, 200, result)
}
