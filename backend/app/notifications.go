package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"firebase.google.com/go/v4/messaging"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Notification struct {
	ID      string            `json:"notification_id" bson:"notification_id"`
	UserID  string            `json:"-" bson:"user_id"`
	Type    string            `json:"type" bson:"type"`
	Title   string            `json:"title" bson:"title"`
	Body    string            `json:"body" bson:"body"`
	Data    map[string]string `json:"data" bson:"data"`
	Read    bool              `json:"read" bson:"read"`
	Created time.Time         `json:"created_at" bson:"created_at"`
}
type NotificationRepository interface {
	Notifications(context.Context, string, int, bool) ([]Notification, int64, error)
	ChangeNotifications(context.Context, string, string, string) (int64, error)
}

func (m mongoUsers) Notifications(ctx context.Context, user string, limit int, unread bool) ([]Notification, int64, error) {
	collection := m.collection.Database().Collection("notifications")
	query := bson.M{"user_id": user}
	if unread {
		query["read"] = false
	}
	pageQuery(ctx, query, "created_at", "notification_id")
	cursor, err := collection.Find(ctx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "notification_id", Value: -1}}).SetLimit(pageLimit(ctx, limit)))
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	records := []Notification{}
	if err = cursor.All(ctx, &records); err != nil {
		return nil, 0, err
	}
	records = trimPage(ctx, records, func(notification Notification) (time.Time, string) { return notification.Created, notification.ID })
	count, err := collection.CountDocuments(ctx, bson.M{"user_id": user, "read": false})
	return records, count, err
}
func (m mongoUsers) ChangeNotifications(ctx context.Context, user, id, action string) (int64, error) {
	collection := m.collection.Database().Collection("notifications")
	query := bson.M{"user_id": user}
	if id != "" {
		query["notification_id"] = id
	}
	if action == "delete" {

		session, err := collection.Database().Client().StartSession()
		if err != nil {
			return 0, err
		}
		defer session.EndSession(ctx)
		value, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
			result, err := collection.DeleteMany(tx, query)
			if err != nil {
				return nil, err
			}
			if result.DeletedCount > 0 {
				// A completed outbox invalidation has no push delivery work. Keep it atomic
				// with deletion because a deleted document no longer identifies its owner.
				_, err = collection.Database().Collection("notification_outbox").InsertOne(tx, bson.M{"type": "inbox_changed", "target_user_id": user, "status": "completed", "created_at": time.Now().UTC()})
				if err != nil {
					return nil, err
				}
			}
			return result.DeletedCount, nil
		})
		if err != nil {
			return 0, err
		}
		return value.(int64), nil
	}
	result, err := collection.UpdateMany(ctx, query, bson.M{"$set": bson.M{"read": true}})
	if err != nil {
		return 0, err
	}
	return result.MatchedCount, nil
}
func (a *AuthDependencies) notificationHandler(action string) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		if !user.Onboarded {
			writeJSON(w, 403, map[string]string{"detail": "Complete onboarding first"})
			return
		}
		repository, ok := a.Users.(NotificationRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Notification storage unavailable"})
			return
		}
		if action == "list" {
			paged, valid := preparePage(w, r, "notifications:"+user.ID, 50)
			if !valid {
				return
			}
			r = paged
			limit := requestPage(r.Context()).Limit
			rows, count, err := repository.Notifications(r.Context(), user.ID, limit, r.URL.Query().Get("unread_only") == "true")
			if err != nil {
				writeJSON(w, 503, map[string]string{"detail": "Unable to load notifications"})
				return
			}
			writePageHeader(w, r)
			writeJSON(w, 200, map[string]any{"notifications": rows, "unread_count": count})
			return
		}
		count, err := repository.ChangeNotifications(r.Context(), user.ID, r.PathValue("notification_id"), action)
		if err != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to update notifications"})
			return
		}
		writeJSON(w, 200, map[string]any{"success": true, "count": count, "message": "Notifications updated"})
	}
}

type NotificationWorker interface{ RunNotifications(context.Context) }
type deliveryEvent struct {
	EngagementID string          `bson:"engagement_id"`
	Expires      time.Time       `bson:"expires_at"`
	ReportID     string          `bson:"report_id"`
	RideTime     time.Time       `bson:"ride_time"`
	TicketID     string          `bson:"ticket_id"`
	Target       string          `bson:"target_user_id"`
	Departure    time.Time       `bson:"departure"`
	ID           any             `bson:"_id"`
	Type         string          `bson:"type"`
	GroupID      string          `bson:"group_id"`
	Actor        string          `bson:"user_id"`
	Attempts     int             `bson:"attempts"`
	Lease        string          `bson:"lease_token"`
	Delivered    map[string]bool `bson:"delivered"`
}

func stableID(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// Keep scheduling and delivery independent. One reserved delivery worker reads
// reminders only, while three general workers also prioritize them. Ordinary
// provider latency/backlog must not consume every reminder delivery slot.
func (m mongoUsers) RunNotifications(ctx context.Context) {
	var workers sync.WaitGroup
	workers.Add(6)
	go func() { defer workers.Done(); m.runReminderSchedule(ctx) }()
	go func() { defer workers.Done(); m.runEngagementSchedule(ctx) }()
	for worker := 0; worker < 4; worker++ {
		go func() {
			defer workers.Done()
			m.runNotificationDelivery(ctx, worker)
		}()
	}
	workers.Wait()
}
func (m mongoUsers) runReminderSchedule(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		work, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := m.scheduleReminders(work)
		cancel()
		m.recordWorker(ctx, "reminder_schedule", err)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (m mongoUsers) runNotificationDelivery(ctx context.Context, worker int) {
	name := fmt.Sprintf("notifications_%d", worker)
	if worker == 0 {
		name = "notifications_reminders"
	}
	heartbeat := time.Time{}
	lastFailed := false
	for ctx.Err() == nil {
		work, cancel := context.WithTimeout(ctx, 60*time.Second)
		processed, err := m.deliverEvent(work, worker == 0)
		cancel()
		if err != nil || lastFailed || time.Since(heartbeat) >= 30*time.Second {
			m.recordWorker(ctx, name, err)
			heartbeat = time.Now()
		}
		lastFailed = err != nil
		if processed && err == nil {
			continue
		}
		// Empty queues/errors back off without polling in a tight loop. Claimed
		// work drains immediately; idempotency and leases work across replicas.
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func (m mongoUsers) scheduleReminders(ctx context.Context) error {
	now := time.Now().UTC()
	outbox := m.collection.Database().Collection("notification_outbox")
	if _, err := outbox.UpdateMany(ctx, bson.M{"status": "processing", "attempts": bson.M{"$gte": 8}, "lease_until": bson.M{"$lte": now}}, bson.M{"$set": bson.M{"status": "failed", "last_error": "Delivery lease expired at retry limit"}}); err != nil {
		return err
	}
	groups := m.collection.Database().Collection("ride_groups")
	window := bson.M{"$gt": now, "$lte": now.Add(15 * time.Minute)}
	cursor, err := groups.Find(ctx, bson.M{
		"status": bson.M{"$in": []string{"active", "in_progress"}},
		"$or": bson.A{
			bson.M{"reminder_schedule": bson.M{"$elemMatch": bson.M{"ride_time": window}}},
			bson.M{"reminder_schedule.0": bson.M{"$exists": false}, "departure_start": window},
		},
	}, options.Find().SetSort(bson.D{{Key: "departure_start", Value: 1}, {Key: "group_id", Value: 1}}).SetBatchSize(100).SetProjection(bson.M{
		"group_id": 1, "departure_start": 1, "pickup_summary": 1, "status": 1, "mode": 1, "reminder_schedule": 1,
		"members.user_id": 1, "members.finished": 1,
	}))
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	models := []mongo.WriteModel{}
	flush := func() error {
		if len(models) == 0 {
			return nil
		}
		_, err := outbox.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
		models = models[:0]
		return err
	}
	for cursor.Next(ctx) {
		var group GroupRecord
		if err = cursor.Decode(&group); err != nil {
			return err
		}
		if len(group.Members) < 2 {
			continue
		}
		for _, member := range group.Members {
			planned, ok := reminderForMember(group, member.UserID)
			if !ok || !planned.RideTime.After(now) || planned.RideTime.After(now.Add(15*time.Minute)) {
				continue
			}
			id := reminderEventID(group.ID, group.Departure, planned.RideTime, member.UserID)
			models = append(models, mongo.NewUpdateOneModel().SetFilter(bson.M{"_id": id}).SetUpdate(bson.M{"$setOnInsert": bson.M{
				"type": "ride_reminder_15", "priority": 100, "group_id": group.ID, "departure": group.Departure, "ride_time": planned.RideTime,
				"target_user_id": member.UserID, "status": "pending", "attempts": 0, "created_at": now,
			}}).SetUpsert(true))
			if len(models) >= 400 {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
	if err := cursor.Err(); err != nil {
		return err
	}
	return flush()
}

func (m mongoUsers) deliverEvent(ctx context.Context, remindersOnly bool) (bool, error) {
	outbox := m.collection.Database().Collection("notification_outbox")
	now := time.Now().UTC()
	lease := stableID(fmt.Sprintf("%d:%p", now.UnixNano(), &now))
	query := bson.M{"status": bson.M{"$in": []string{"pending", "processing"}}, "attempts": bson.M{"$lt": 8}, "$and": bson.A{
		bson.M{"$or": bson.A{bson.M{"lease_until": bson.M{"$lte": now}}, bson.M{"lease_until": bson.M{"$exists": false}}}},
		bson.M{"$or": bson.A{bson.M{"next_attempt_at": bson.M{"$lte": now}}, bson.M{"next_attempt_at": bson.M{"$exists": false}}}},
	}}
	if remindersOnly {
		query["type"] = "ride_reminder_15"
	}
	var event deliveryEvent
	err := outbox.FindOneAndUpdate(ctx, query, bson.M{"$set": bson.M{"status": "processing", "lease_token": lease, "lease_until": now.Add(90 * time.Second)}, "$inc": bson.M{"attempts": 1}}, options.FindOneAndUpdate().SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}).SetReturnDocument(options.After)).Decode(&event)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	filter := bson.M{"_id": event.ID, "lease_token": lease}
	deliverErr := m.deliverRecipients(ctx, event)
	status := "sent"
	update := bson.M{"status": status, "processed_at": time.Now().UTC()}
	if deliverErr != nil {
		status = "pending"
		if event.Attempts >= 8 {
			status = "failed"
		}
		update = bson.M{"status": status, "next_attempt_at": time.Now().Add(time.Duration(1<<min(event.Attempts, 8)) * time.Second), "last_error": "Delivery failed"}
	}
	_, err = outbox.UpdateOne(ctx, filter, bson.M{"$set": update, "$unset": bson.M{"lease_until": "", "lease_token": ""}})
	if err != nil {
		return true, err
	}
	return true, deliverErr
}
func (m mongoUsers) deliverRecipients(ctx context.Context, event deliveryEvent) error {
	db := m.collection.Database()
	var group GroupRecord
	if event.GroupID == "" {
		if event.Target == "" {
			return nil
		}
		group.Members = []GroupMemberRecord{{UserID: event.Target}}
	} else {
		if err := db.Collection("ride_groups").FindOne(ctx, bson.M{"group_id": event.GroupID}, options.FindOne().SetProjection(bson.M{
			"group_id": 1, "status": 1, "mode": 1, "departure_start": 1, "pickup_summary": 1, "reminder_schedule": 1, "members.user_id": 1, "members.finished": 1,
		})).Decode(&group); errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		} else if err != nil {
			return err
		}
	}
	engagement := event.Type == "engagement_reminder"
	var nudge EngagementMessage
	if engagement {
		var ok bool
		var err error
		nudge, ok, err = m.currentEngagement(ctx, event)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	reminder := event.Type == "ride_reminder_15"
	var planned MemberReminder
	if reminder {
		var ok bool
		planned, ok = matchingReminder(group, event, time.Now())
		if !ok {
			return nil
		}
	}
	title, body, kind := "Ride group updated", "Open ORIX for the latest group details.", "system"
	data := map[string]string{"type": event.Type, "group_id": group.ID}
	if engagement {
		title = nudge.Title
		body = nudge.Body
		kind = "engagement_reminder"
		data["action"] = "open_home"
	}
	if event.Type == "account_restored" {
		title = "Account access restored"
		body = "You can return to ORIX. Cancelled trips have not been restarted."
	}
	if event.Type == "support_reply" {
		title = "Support ticket updated"
		body = "Open your support ticket to read the update."
		data["ticket_id"] = event.TicketID
	}
	if event.Type == "report_reply" {
		title = "Reply from Orix moderation"
		body = "Open your private report conversation to read the reply."
		kind = "report_action"
		data["report_id"] = event.ReportID
	}
	if event.Type == "report_resolved" {
		data["report_id"] = event.ReportID
		title = "Your report was reviewed"
		body = "Open your reports to see its status."
		kind = "report_action"
	}
	if event.Type == "moderation_warning" {
		title = "Account safety notice"
		body = "A moderator has issued a warning. Review the community rules before your next trip."
		kind = "report_action"
	}
	if event.Type == "group_rejected" {
		title = "Group matching restarted"
		body = "Your previous group changed. Open Orix to see your current match."
	}
	if event.Type == "new_chat_message" {
		title = "New group message"
		body = "Open your ride group to read the message."
		kind = "chat_message"
	}
	if reminder {
		title = fmt.Sprintf("Your ride is in %d minutes", max(1, int(math.Ceil(time.Until(planned.RideTime).Minutes()))))
		body = "Check your meeting point and get ready."
		data["type"] = "RIDE_ALARM"
		data["action"] = "show_alarm"
		data["ride_time"] = planned.RideTime.Format(time.RFC3339Nano)
		data["pickup_summary"] = planned.Pickup
		data["title"] = title
		data["body"] = body
	}
	for _, member := range group.Members {
		if event.Target != "" && member.UserID != event.Target {
			continue
		}
		if event.Type == "new_chat_message" {
			blocked, err := m.BlockedPeers(ctx, member.UserID)
			if err != nil {
				return err
			}
			if blocked[event.Actor] {
				continue
			}
		}
		if event.Type == "new_chat_message" && member.UserID == event.Actor {
			continue
		}
		recipientKey := stableID(member.UserID)
		if event.Delivered[recipientKey] {
			continue
		}
		var user struct {
			ID                string          `bson:"user_id"`
			FirebaseUID       string          `bson:"firebase_uid"`
			Token             string          `bson:"fcm_token"`
			Status            string          `bson:"status"`
			Preferences       map[string]bool `bson:"notification_preferences"`
			SuspensionExpires *time.Time      `bson:"suspension_expires_at"`
		}
		if err := m.collection.FindOne(ctx, bson.M{"user_id": member.UserID}).Decode(&user); errors.Is(err, mongo.ErrNoDocuments) {
			continue
		} else if err != nil {
			return err
		}
		if accountRestricted(User{Status: user.Status, SuspensionExpires: user.SuspensionExpires}) {
			continue
		}
		if engagement && !user.Preferences["promotions"] {
			continue
		}
		preference := "group_updates"
		if engagement {
			preference = "promotions"
		}
		if reminder {
			preference = "ride_reminders"
		} else if kind == "chat_message" {
			preference = "chat_messages"
		}
		if reminder {
			// Recheck current membership, completion and the persisted personal pickup
			// schedule immediately before inbox/push work. Accepted pushes cannot be recalled.
			var current GroupRecord
			err := db.Collection("ride_groups").FindOne(ctx, bson.M{"group_id": group.ID, "members.user_id": member.UserID}, options.FindOne().SetProjection(bson.M{
				"status": 1, "mode": 1, "departure_start": 1, "pickup_summary": 1, "reminder_schedule": 1, "members.user_id": 1, "members.finished": 1,
			})).Decode(&current)
			if errors.Is(err, mongo.ErrNoDocuments) {
				continue
			}
			if err != nil {
				return err
			}
			fresh, ok := matchingReminder(current, event, time.Now())
			if !ok {
				continue
			}
			planned = fresh
			data["pickup_summary"] = fresh.Pickup
		}
		id := stableID(fmt.Sprintf("%v:%s", event.ID, member.UserID))
		recipientData := maps.Clone(data)
		recipientData["notification_id"] = id
		recipientData["recipient_user_id"] = user.ID
		recipientData["recipient_firebase_uid"] = user.FirebaseUID
		notification := Notification{id, user.ID, kind, title, body, recipientData, false, time.Now().UTC()}
		if _, err := db.Collection("notifications").UpdateOne(ctx, bson.M{"notification_id": id}, bson.M{"$setOnInsert": notification}, options.UpdateOne().SetUpsert(true)); err != nil {
			return err
		}
		// Re-read the destination immediately before push delivery. A sign-out,
		// reassignment or preference change may have occurred since the inbox write.
		var current struct {
			Token             string          `bson:"fcm_token"`
			Status            string          `bson:"status"`
			SuspensionExpires *time.Time      `bson:"suspension_expires_at"`
			Preferences       map[string]bool `bson:"notification_preferences"`
		}
		err := m.collection.FindOne(ctx, bson.M{"user_id": user.ID}, options.FindOne().SetProjection(bson.M{"fcm_token": 1, "status": 1, "suspension_expires_at": 1, "notification_preferences": 1})).Decode(&current)
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			return err
		}
		enabled, present := current.Preferences[preference]
		if engagement && (!present || !enabled || !time.Now().Before(event.Expires)) {
			continue
		}
		if err != nil || current.Token != user.Token || accountRestricted(User{Status: current.Status, SuspensionExpires: current.SuspensionExpires}) {
			user.Token = ""
		}
		androidKey := "android_" + recipientKey
		if user.Token != "" && m.push != nil && (!present || enabled) && !event.Delivered[androidKey] {
			android := &messaging.AndroidConfig{Priority: "high", CollapseKey: id, Notification: &messaging.AndroidNotification{Icon: "ic_notification", Color: "#222222", ChannelID: "high_importance_channel", Tag: id}}
			message := &messaging.Message{Token: user.Token, Data: recipientData, Notification: &messaging.Notification{Title: title, Body: body}, Android: android}
			if engagement {
				ttl := time.Until(event.Expires)
				android.Priority = "normal"
				android.TTL = &ttl
				android.Notification.ChannelID = "orix_updates"
			}
			if reminder {
				message.Notification = nil
				android.Notification = nil
				// Allow for the bounded send request before a relative Android TTL
				// starts at provider acceptance. Near-departure reminders are skipped.
				ttl := min(time.Minute, time.Until(planned.RideTime)-10*time.Second)
				if ttl <= 0 {
					// One expired pickup must not suppress later recipients.
					continue
				}
				android.TTL = &ttl
				android.CollapseKey = stableID("ride-reminder:" + group.ID + ":" + user.ID)
				message.APNS = &messaging.APNSConfig{Headers: map[string]string{
					"apns-priority": "10", "apns-push-type": "alert", "apns-expiration": strconv.FormatInt(time.Now().Add(ttl).Unix(), 10),
					"apns-collapse-id": android.CollapseKey,
				}, Payload: &messaging.APNSPayload{Aps: &messaging.Aps{Alert: &messaging.ApsAlert{Title: title, Body: body}, Sound: "default"}}}
			}
			sendContext, cancel := context.WithTimeout(ctx, 10*time.Second)
			_, sendErr := m.push.Send(sendContext, message)
			cancel()
			if err := sendErr; err != nil {
				if messaging.IsRegistrationTokenNotRegistered(err) {
					if err := m.releasePushToken(ctx, user.ID, user.Token); err != nil {
						return err
					}
				} else {
					return err
				}
			}
		}
		if user.Token != "" && m.push != nil && (!present || enabled) && !event.Delivered[androidKey] {
			result, err := db.Collection("notification_outbox").UpdateOne(ctx, bson.M{"_id": event.ID, "lease_token": event.Lease}, bson.M{"$set": bson.M{"delivered." + androidKey: true}})
			if err != nil {
				return err
			}
			if result.MatchedCount != 1 {
				return errors.New("notification lease expired")
			}
		}
		if m.push != nil && (!present || enabled) && err == nil && !accountRestricted(User{Status: current.Status, SuspensionExpires: current.SuspensionExpires}) {
			cursor, err := db.Collection("web_push_tokens").Find(ctx, bson.M{"user_id": user.ID, "updated_at": bson.M{"$gte": time.Now().UTC().Add(-30 * 24 * time.Hour)}}, options.Find().SetLimit(10))
			if err != nil {
				return err
			}
			var devices []struct {
				ID    string `bson:"_id"`
				Token string `bson:"token"`
			}
			err = cursor.All(ctx, &devices)
			cursor.Close(ctx)
			if err != nil {
				return err
			}
			for _, device := range devices {
				// Account state and preferences may change while another device
				// is being sent; re-read before every individual delivery.
				var latest struct {
					Status            string          `bson:"status"`
					SuspensionExpires *time.Time      `bson:"suspension_expires_at"`
					Preferences       map[string]bool `bson:"notification_preferences"`
				}
				if err := m.collection.FindOne(ctx, bson.M{"user_id": user.ID}, options.FindOne().SetProjection(bson.M{"status": 1, "suspension_expires_at": 1, "notification_preferences": 1})).Decode(&latest); err != nil {
					if errors.Is(err, mongo.ErrNoDocuments) {
						break
					}
					return err
				}
				if accountRestricted(User{Status: latest.Status, SuspensionExpires: latest.SuspensionExpires}) {
					break
				}
				if allowed, exists := latest.Preferences[preference]; (exists && !allowed) || (engagement && (!exists || !time.Now().Before(event.Expires))) {
					break
				}
				key := "web_" + device.ID
				if event.Delivered[key] {
					continue
				}
				// Recheck ownership immediately before delivery; no private
				// route, message text or Firebase identifier enters the payload.
				count, err := db.Collection("web_push_tokens").CountDocuments(ctx, bson.M{"_id": device.ID, "user_id": user.ID, "token": device.Token})
				if err != nil {
					return err
				}
				if count == 0 {
					continue
				}
				message := &messaging.Message{Token: device.Token, Data: map[string]string{"notification_id": id, "type": "inbox_update"}}
				if engagement {
					message.Data["type"] = "engagement_reminder"
					message.Data["title"] = title
					message.Data["body"] = body
					message.Webpush = &messaging.WebpushConfig{Headers: map[string]string{"TTL": strconv.Itoa(max(0, int(time.Until(event.Expires).Seconds()))), "Urgency": "low"}}
				}
				sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				_, sendErr := m.push.Send(sendCtx, message)
				cancel()
				if sendErr != nil {
					if messaging.IsRegistrationTokenNotRegistered(sendErr) {
						if err := m.UnregisterWebPush(ctx, user.ID, device.Token); err != nil {
							return err
						}
					} else {
						return sendErr
					}
				}
				result, err := db.Collection("notification_outbox").UpdateOne(ctx, bson.M{"_id": event.ID, "lease_token": event.Lease}, bson.M{"$set": bson.M{"delivered." + key: true}})
				if err != nil {
					return err
				}
				if result.MatchedCount != 1 {
					return errors.New("notification lease expired")
				}
			}
		}
		if _, err := db.Collection("notification_outbox").UpdateOne(ctx, bson.M{"_id": event.ID, "lease_token": event.Lease}, bson.M{"$set": bson.M{"delivered." + recipientKey: true}}); err != nil {
			return err
		}
	}
	return nil
}
