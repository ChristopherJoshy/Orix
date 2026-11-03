package server

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Editorial content, never simulated journeys or claims that a match exists.
type EngagementMessage struct {
	ID      string `json:"id" bson:"id"`
	Title   string `json:"title" bson:"title"`
	Body    string `json:"body" bson:"body"`
	Enabled bool   `json:"enabled" bson:"enabled"`
}
type EngagementConfig struct {
	Version  int64               `json:"version" bson:"version"`
	Enabled  bool                `json:"enabled" bson:"enabled"`
	Messages []EngagementMessage `json:"messages" bson:"messages"`
}
type EngagementRepository interface {
	GetEngagement(context.Context) (EngagementConfig, error)
	SaveEngagement(context.Context, string, EngagementConfig) (EngagementConfig, error)
}

var errEngagementConflict = errors.New("engagement config changed")
var engagementIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func defaultEngagement() EngagementConfig {
	return EngagementConfig{Version: 1, Enabled: true, Messages: []EngagementMessage{
		{"chaya", "Chaya break kazhinjo?", "Next stop evideya? Plan your journey on ORIX before the next chaya break.", true},
		{"scene", "Scene contra aakkanda.", "Last-minute travel plans? Oru plan undaakkam. Open ORIX.", true},
		{"evide", "Evide, evide?", "Destination ariyam. Company nokkiyaalo? Plan a journey on ORIX.", true},
		{"povaam", "Enna pinne, povaam?", "Bag ready. Playlist ready. Route plan cheytho?", true},
		{"auto", "Auto varumo?", "Athinu munpe, plan ready aakkam. Check your route on ORIX.", true},
		{"mazha", "Mazha peythaalum plan venam.", "Umbrella edutho? Your next journey can use a little planning too.", true},
		{"naale", "Naale nokkaam ennaano?", "Tomorrow's trip can be today's plan. Oru two-minute kaaryam.", true},
		{"pokam", "Pokaan neram aayo?", "Keys, phone, wallet. Pinne route plan. ORIX is here when you need it.", true},
		{"sugham", "Sugham alle?", "Just checking in. Next journey undenkil, ORIX-il plan cheyyaam.", true},
		{"set", "Plan set aakkiyaalo?", "Destination first. Baakki pinne. Open ORIX for your next journey.", true},
		{"bus", "Bus stop philosophy venda.", "A little planning before you leave. Athre ullu. Check ORIX.", true},
		{"oru", "Oru journey plan cheyyaam?", "Long story venda. Just your route, your time, your preferences.", true},
		{"traffic", "Traffic-inu script illa.", "At least your travel plan can have one. Open ORIX before you leave.", true},
		{"company", "Company kittiyaal kollam.", "Heading somewhere soon? Plan on ORIX and look for people on your route.", true},
		{"nammal", "Nammal ivide undu.", "No dramatic comeback needed. Open ORIX when your next trip is on the cards.", true},
		{"route", "Route marannittillaallo?", "Your next destination is a good place to start. Plan it on ORIX.", true},
		{"seat", "Oru seat spare undo?", "Driving somewhere? You can offer a lift on your route through ORIX.", true},
		{"ready", "Ready, steady... evidekku?", "Pick a destination first. ORIX can help you plan the journey.", true},
		{"pinne", "Pinne kaanaam?", "Sure. But first, next trip plan cheythu vekkaam. Open ORIX.", true},
		{"vazhi", "Vazhi ariyaam. Plan undo?", "A familiar route deserves a proper plan too. Orix-il nokkiyaalo?", true},
	}}
}
func validateEngagement(config EngagementConfig) bool {
	if config.Version < 1 || len(config.Messages) < 1 || len(config.Messages) > 200 {
		return false
	}
	seen := map[string]bool{}
	for _, message := range config.Messages {
		if !engagementIDPattern.MatchString(message.ID) || seen[message.ID] {
			return false
		}
		seen[message.ID] = true
		if strings.TrimSpace(message.Title) == "" || strings.TrimSpace(message.Body) == "" || utf8.RuneCountInString(message.Title) > 80 || utf8.RuneCountInString(message.Body) > 240 {
			return false
		}
		// Notifications are one-line authored text, not URLs or control sequences.
		if strings.ContainsAny(message.Title+message.Body, "\x00\r\n") {
			return false
		}
	}
	return true
}

// India-first schedule: 08:00, 12:00, 16:00, 20:00 IST; no nighttime catch-up.
// A thirty-minute window bounds retries and prevents old nudges arriving later.
func engagementSlot(now time.Time) (time.Time, time.Time, bool) {
	local := now.In(time.FixedZone("IST", 19800))
	if local.Hour() < 8 || local.Hour() > 20 || (local.Hour()-8)%4 != 0 || local.Minute() >= 30 {
		return time.Time{}, time.Time{}, false
	}
	start := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, local.Location()).UTC()
	return start, start.Add(30 * time.Minute), true
}
func chooseEngagement(config EngagementConfig, slot time.Time, userID string) (EngagementMessage, bool) {
	enabled := []EngagementMessage{}
	for _, message := range config.Messages {
		if message.Enabled {
			enabled = append(enabled, message)
		}
	}
	if !config.Enabled || len(enabled) == 0 {
		return EngagementMessage{}, false
	}
	// Advance each slot; stable user offset spreads variants without random retry changes.
	hash := stableID(userID)
	offset := int64(0)
	for _, c := range hash[:8] {
		offset = offset*31 + int64(c)
	}
	index := (slot.Unix()/int64((4*time.Hour)/time.Second) + offset) % int64(len(enabled))
	return enabled[index], true
}
func (m mongoUsers) GetEngagement(ctx context.Context) (EngagementConfig, error) {
	collection := m.collection.Database().Collection("engagement_config")
	var config EngagementConfig
	err := collection.FindOne(ctx, bson.M{"_id": "reminders_v1"}).Decode(&config)
	if errors.Is(err, mongo.ErrNoDocuments) {
		defaults := defaultEngagement()
		_, err = collection.UpdateOne(ctx, bson.M{"_id": "reminders_v1"}, bson.M{"$setOnInsert": defaults}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return config, err
		}
		err = collection.FindOne(ctx, bson.M{"_id": "reminders_v1"}).Decode(&config)
	}
	return config, err
}
func (m mongoUsers) SaveEngagement(ctx context.Context, actor string, config EngagementConfig) (EngagementConfig, error) {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return config, err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		next := config
		next.Version++
		db := m.collection.Database()
		result, err := db.Collection("engagement_config").UpdateOne(tx, bson.M{"_id": "reminders_v1", "version": config.Version}, bson.M{"$set": next})
		if err != nil {
			return nil, err
		}
		if result.MatchedCount != 1 {
			return nil, errEngagementConflict
		}
		_, err = WriteAudit(tx, db, bson.M{"actor_id": actor, "action": "update_engagement_messages", "target_id": "reminders_v1", "created_at": time.Now().UTC()})
		return nil, err
	})
	if err == nil {
		config.Version++
	}
	return config, err
}
func (a *AuthDependencies) engagementSettings(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.Users.(EngagementRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Reminder settings unavailable"})
		return
	}
	if r.Method == http.MethodGet {
		config, err := repository.GetEngagement(r.Context())
		if err != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to load reminder settings"})
			return
		}
		writeJSON(w, 200, config)
		return
	}
	var config EngagementConfig
	if decodeBody(r, &config) != nil || !validateEngagement(config) {
		writeJSON(w, 422, map[string]string{"detail": "Invalid reminder messages"})
		return
	}
	saved, err := repository.SaveEngagement(r.Context(), user.ID, config)
	if errors.Is(err, errEngagementConflict) {
		writeJSON(w, 409, map[string]string{"detail": "Settings changed; reload before saving"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to save reminder settings"})
		return
	}
	writeJSON(w, 200, saved)
}
func (m mongoUsers) runEngagementSchedule(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for ctx.Err() == nil {
		work, cancel := context.WithTimeout(ctx, 45*time.Second)
		err := m.scheduleEngagement(work, time.Now().UTC())
		cancel()
		m.recordWorker(ctx, "engagement_schedule", err)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (m mongoUsers) scheduleEngagement(ctx context.Context, now time.Time) error {
	start, expires, active := engagementSlot(now)
	if !active {
		return nil
	}
	config, err := m.GetEngagement(ctx)
	if err != nil {
		return err
	}
	if !config.Enabled {
		return nil
	}
	// Explicit opt-in is required. Missing promotion preferences remain off.
	query := bson.M{"onboarding_completed": true, "status": bson.M{"$nin": []string{"banned", "suspended", "deleted", "pending_deletion"}}, "notification_preferences.promotions": true,
		"$or": bson.A{bson.M{"last_engagement_at": bson.M{"$lte": now.Add(-4 * time.Hour)}}, bson.M{"last_engagement_at": bson.M{"$exists": false}}}}
	cursor, err := m.collection.Find(ctx, query, options.Find().SetBatchSize(100).SetProjection(bson.M{"user_id": 1}))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var user struct {
			ID string `bson:"user_id"`
		}
		if err = cursor.Decode(&user); err != nil {
			return err
		}
		message, ok := chooseEngagement(config, start, user.ID)
		if !ok {
			return nil
		}
		session, err := m.collection.Database().Client().StartSession()
		if err != nil {
			return err
		}
		_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
			// Reservation and cooldown commit together; duplicate replicas cannot enqueue twice.
			filter := bson.M{}
			for k, v := range query {
				filter[k] = v
			}
			filter["user_id"] = user.ID
			updated, err := m.collection.UpdateOne(tx, filter, bson.M{"$set": bson.M{"last_engagement_at": now}})
			if err != nil {
				return nil, err
			}
			if updated.MatchedCount != 1 {
				return nil, nil
			}
			id := "engagement:" + stableID(user.ID+":"+start.Format(time.RFC3339))
			_, err = m.collection.Database().Collection("notification_outbox").UpdateOne(tx, bson.M{"_id": id}, bson.M{"$setOnInsert": bson.M{"type": "engagement_reminder", "target_user_id": user.ID, "engagement_id": message.ID, "expires_at": expires, "status": "pending", "attempts": 0, "created_at": now}}, options.UpdateOne().SetUpsert(true))
			return nil, err
		})
		session.EndSession(ctx)
		if err != nil {
			return err
		}
	}
	return cursor.Err()
}
func (m mongoUsers) currentEngagement(ctx context.Context, event deliveryEvent) (EngagementMessage, bool, error) {
	_, _, active := engagementSlot(time.Now().UTC())
	if !active || event.Expires.IsZero() || !time.Now().Before(event.Expires) {
		return EngagementMessage{}, false, nil
	}
	config, err := m.GetEngagement(ctx)
	if err != nil {
		return EngagementMessage{}, false, err
	}
	if config.Enabled {
		for _, message := range config.Messages {
			if message.ID == event.EngagementID && message.Enabled {
				return message, true, nil
			}
		}
	}
	return EngagementMessage{}, false, nil
}
