package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type RealtimeEvent struct {
	Recipients []string
	GroupID    string
	Type       string
	Message    *ChatMessage
	AdminsOnly bool
	Broadcast  bool
}
type RealtimeSource interface {
	StreamEvents(context.Context, func(RealtimeEvent)) error
}

func (m mongoUsers) StreamEvents(ctx context.Context, publish func(RealtimeEvent)) error {
	// Database-level invalidations cover state changed from another device as well
	// as worker-produced outbox inserts. Only opaque invalidations leave this layer.
	pipeline := bson.A{bson.M{"$match": bson.M{
		"operationType": bson.M{"$in": bson.A{"insert", "update", "replace"}},
		"ns.coll":       bson.M{"$in": bson.A{"notification_outbox", "ride_requests", "notifications", "support_tickets", m.collection.Name()}},
	}}}
	stream, err := m.collection.Database().Watch(ctx, pipeline, options.ChangeStream().SetFullDocument(options.UpdateLookup))
	if err != nil {
		return err
	}
	defer stream.Close(ctx)
	publish(RealtimeEvent{Type: "resync", Broadcast: true})
	for stream.Next(ctx) {
		var change struct {
			Operation string `bson:"operationType"`
			Namespace struct {
				Collection string `bson:"coll"`
			} `bson:"ns"`
			Document struct {
				UserID    string `bson:"user_id"`
				Type      string `bson:"type"`
				GroupID   string `bson:"group_id"`
				MessageID string `bson:"message_id"`
				Target    string `bson:"target_user_id"`
				ReportID  string `bson:"report_id"`
			} `bson:"fullDocument"`
		}
		if err = stream.Decode(&change); err != nil {
			return err
		}
		doc := change.Document
		if change.Namespace.Collection != "notification_outbox" {
			eventType := resourceEventType(change.Namespace.Collection)
			if change.Namespace.Collection == m.collection.Name() {
				eventType = "profile_update"
			}
			if eventType != "" && doc.UserID != "" {
				publish(RealtimeEvent{Type: eventType, Recipients: []string{doc.UserID}})
			}
			if change.Namespace.Collection == "support_tickets" {
				if err := m.publishAdminInvalidation(ctx, publish, "support_update", doc.UserID); err != nil {
					return err
				}
			}
			if change.Namespace.Collection == m.collection.Name() && doc.UserID != "" {
				cursor, err := m.collection.Database().Collection("ride_groups").Find(ctx, bson.M{"members.user_id": doc.UserID, "status": bson.M{"$in": bson.A{"pending", "confirming", "active", "in_progress"}}}, options.Find().SetLimit(100).SetProjection(bson.M{"group_id": 1, "members.user_id": 1}))
				if err != nil {
					return err
				}
				var groups []CandidateGroup
				err = cursor.All(ctx, &groups)
				cursor.Close(ctx)
				if err != nil {
					return err
				}
				for _, group := range groups {
					recipients := make([]string, 0, len(group.Members))
					for _, member := range group.Members {
						recipients = append(recipients, member.UserID)
					}
					publish(RealtimeEvent{Type: "member_update", GroupID: group.ID, Recipients: recipients})
				}
			}

			continue
		}
		// Outbox delivery bookkeeping is not itself a new domain event.
		if change.Operation != "insert" {
			continue
		}
		if doc.GroupID == "" {
			if doc.Target != "" {
				publish(RealtimeEvent{Type: "notification_update", Recipients: []string{doc.Target}})
			}
			if doc.Type == "report_created" || doc.Type == "report_followup" {
				var report ReportRecord
				if m.collection.Database().Collection("reports").FindOne(ctx, bson.M{"report_id": doc.ReportID}).Decode(&report) != nil {
					continue
				}
				cursor, err := m.collection.Find(ctx, bson.M{"role": "admin", "user_id": bson.M{"$nin": bson.A{report.Reporter, report.Accused}}}, options.Find().SetLimit(100).SetProjection(bson.M{"user_id": 1}))
				if err != nil {
					return err
				}
				var admins []User
				err = cursor.All(ctx, &admins)
				cursor.Close(ctx)
				if err != nil {
					return err
				}
				ids := make([]string, 0, len(admins))
				for _, admin := range admins {
					ids = append(ids, admin.ID)
				}
				publish(RealtimeEvent{Type: "report_update", Recipients: ids, AdminsOnly: true})
			}
			continue
		}
		var group CandidateGroup
		if err = m.collection.Database().Collection("ride_groups").FindOne(ctx, bson.M{"group_id": doc.GroupID}).Decode(&group); err != nil {
			continue
		}
		recipients := make([]string, 0, len(group.Members))
		for _, member := range group.Members {
			recipients = append(recipients, member.UserID)
		}
		event := RealtimeEvent{GroupID: doc.GroupID, Type: "readiness_update", Recipients: recipients}
		if doc.Type == "new_chat_message" {
			var message ChatMessage
			if err = m.collection.Database().Collection("chat_messages").FindOne(ctx, bson.M{"message_id": doc.MessageID, "group_id": doc.GroupID}).Decode(&message); err != nil {
				continue
			}
			event.Type = "new_chat_message"
			event.Message = &message
		}
		publish(event)
	}
	return stream.Err()
}

type socketClient struct {
	userID string
	queue  chan RealtimeEvent
	cancel context.CancelFunc
}
type socketHub struct {
	mu      sync.Mutex
	clients map[*socketClient]bool
}

func (h *socketHub) add(client *socketClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	count := 0
	for other := range h.clients {
		if other.userID == client.userID {
			count++
		}
	}
	if len(h.clients) >= 1000 || count >= 4 {
		return false
	}
	h.clients[client] = true
	return true
}
func (h *socketHub) remove(client *socketClient) {
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()
}
func (h *socketHub) publish(event RealtimeEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	allowed := make(map[string]bool, len(event.Recipients))
	for _, id := range event.Recipients {
		allowed[id] = true
	}
	for client := range h.clients {
		if !event.Broadcast && !allowed[client.userID] {
			continue
		}
		select {
		case client.queue <- event:
		default:
			client.cancel()
		}
	}
}
func (h *socketHub) run(ctx context.Context, source RealtimeSource) {
	for ctx.Err() == nil {
		_ = source.StreamEvents(ctx, h.publish)
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func (a *AuthDependencies) socketUser(ctx context.Context, token string) (User, error) {
	if a == nil || a.Verifier == nil || a.Users == nil {
		return User{}, errors.New("unconfigured auth")
	}
	identity, err := a.Verifier.Verify(ctx, token)
	if err != nil {
		return User{}, err
	}
	user, _, err := a.Users.GetOrCreate(ctx, identity)
	if err != nil {
		return User{}, err
	}
	if !user.Onboarded || accountRestricted(user) {
		return User{}, ErrGroupUnavailable
	}
	return user, nil
}
func (a *AuthDependencies) socketHandler(parent context.Context, hub *socketHub, originPatterns []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Native clients use Bearer. Browsers offer an auth subprotocol because
		// their WebSocket API cannot attach custom headers; only orix.v1 is selected.
		token, err := socketToken(r)
		if err != nil {
			writeJSON(w, 401, map[string]string{"detail": "Valid socket authentication required"})
			return
		}

		authCtx, authCancel := context.WithTimeout(r.Context(), 10*time.Second)
		user, err := a.socketUser(authCtx, token)
		authCancel()
		if err != nil {
			writeJSON(w, 401, map[string]string{"detail": "Sign in and complete onboarding before connecting"})
			return
		}
		repository, ok := a.Users.(ChatRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Chat is not configured"})
			return
		}
		groups, ok := a.Users.(MatchRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Groups are not configured"})
			return
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: originPatterns, Subprotocols: []string{"orix.v1"}})
		if err != nil {
			return
		}
		defer conn.CloseNow()
		conn.SetReadLimit(8192)
		ctx, cancel := context.WithTimeout(parent, 55*time.Minute)
		defer cancel()
		client := &socketClient{userID: user.ID, queue: make(chan RealtimeEvent, 32), cancel: cancel}
		if !hub.add(client) {
			_ = conn.Close(websocket.StatusTryAgainLater, "Connection limit reached")
			return
		}
		defer hub.remove(client)
		write := func(data any) error {
			writeCtx, stop := context.WithTimeout(ctx, 10*time.Second)
			defer stop()
			return wsjson.Write(writeCtx, conn, data)
		}
		if write(map[string]any{"type": "connected", "user_id": user.ID, "timestamp": time.Now().UTC()}) != nil {
			return
		}
		// One event consumer performs fresh account and membership checks before send.
		go func() {
			defer cancel()
			for {
				select {
				case <-ctx.Done():
					return
				case event := <-client.queue:
					check, stop := context.WithTimeout(ctx, 10*time.Second)
					current, err := a.socketUser(check, token)
					if err != nil {
						stop()
						return
					}
					if event.GroupID == "" {
						if event.AdminsOnly && current.Role != "admin" {
							stop()
							continue
						}
						stop()
						if write(map[string]any{"type": event.Type}) != nil {
							return
						}
						continue
					}
					if event.Message != nil {
						if safety, ok := a.Users.(SafetyRepository); ok {
							blocked, blockErr := safety.BlockedPeers(check, current.ID)
							if blockErr != nil || blocked[event.Message.UserID] {
								stop()
								continue
							}
						}
					}
					group, err := groups.GroupForMember(check, current.ID, event.GroupID)
					stop()
					if err != nil {
						continue
					}
					payload := map[string]any{"type": event.Type, "group_id": event.GroupID, "group_status": group.Status}
					if event.Message != nil {
						message := *event.Message
						message.Mine = message.UserID == current.ID
						if !message.Mine {
							message.ClientID = ""
						}
						payload["message"] = message
						payload["message_id"] = message.ID
						payload["content"] = message.Content
						payload["sender_name"] = message.Name
					}
					if group.Status == "active" && event.Message == nil && event.Type == "readiness_update" {
						payload["type"] = "group_finalized"
					}
					if group.Status == "cancelled" || group.Status == "expired" {
						payload["type"] = "group_cancelled"
					}
					if write(payload) != nil {
						return
					}
				}
			}
		}()
		limiter := &limiter{clients: make(map[string]clientWindow), limit: 60, maxClients: 1}
		for {
			readCtx, stop := context.WithTimeout(ctx, 90*time.Second)
			var input struct {
				Type     string `json:"type"`
				GroupID  string `json:"group_id"`
				Content  string `json:"content"`
				ClientID string `json:"client_message_id"`
			}
			err = wsjson.Read(readCtx, conn, &input)
			stop()
			if err != nil {
				return
			}
			if !limiter.allow(user.ID, time.Now()) {
				_ = conn.Close(websocket.StatusPolicyViolation, "Too many messages")
				return
			}
			check, done := context.WithTimeout(ctx, 10*time.Second)
			current, verifyErr := a.socketUser(check, token)
			if verifyErr != nil {
				done()
				_ = conn.Close(websocket.StatusPolicyViolation, "Sign in again")
				return
			}
			switch input.Type {
			case "ping":
				err = write(map[string]any{"type": "pong", "timestamp": time.Now().UTC()})
			case "chat_message":
				content := strings.TrimSpace(input.Content)
				if content == "" || utf8.RuneCountInString(content) > 1000 || input.GroupID == "" || input.ClientID == "" || len(input.GroupID) > 100 || len(input.ClientID) > 100 || strings.ContainsAny(input.ClientID, ":\r\n") {
					err = write(map[string]string{"type": "error", "message": "Invalid chat message", "client_message_id": input.ClientID})
					break
				}
				message, sendErr := repository.SendChat(check, current.ID, input.GroupID, content, input.ClientID)
				if sendErr != nil {
					err = write(map[string]string{"type": "error", "message": "Message could not be sent", "client_message_id": input.ClientID})
					break
				}
				err = write(map[string]any{"type": "chat_sent", "message_id": message.ID, "client_message_id": message.ClientID, "message": message, "timestamp": message.Created})
			default:
				err = write(map[string]string{"type": "error", "message": "Unknown message type"})
			}
			done()
			if err != nil {
				return
			}
		}
	}
}

func socketToken(r *http.Request) (string, error) {
	token := ""
	if header := r.Header.Get("Authorization"); header != "" {
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return "", errors.New("invalid bearer")
		}
		token = parts[1]
	} else {
		version := false
		for _, raw := range strings.Split(strings.Join(r.Header.Values("Sec-WebSocket-Protocol"), ","), ",") {
			offered := strings.TrimSpace(raw)
			if offered == "orix.v1" {
				version = true
			}
			if strings.HasPrefix(offered, "firebase.") {
				if token != "" {
					return "", errors.New("ambiguous socket authentication")
				}
				token = strings.TrimPrefix(offered, "firebase.")
			}
		}
		if !version {
			return "", errors.New("socket protocol required")
		}
	}
	if token == "" || len(token) > 16384 || strings.ContainsAny(token, " \r\n\t") {
		return "", errors.New("invalid socket credential")
	}
	return token, nil
}

func resourceEventType(collection string) string {
	switch collection {
	case "ride_requests":
		return "ride_update"
	case "notifications":
		return "notification_update"
	case "support_tickets":
		return "support_update"
	default:
		return ""
	}
}

// Moderator queue invalidations contain no ticket content and are re-authorized
// against the recipient's current role when delivered by the socket writer.
func (m mongoUsers) publishAdminInvalidation(ctx context.Context, publish func(RealtimeEvent), eventType, exclude string) error {
	cursor, err := m.collection.Find(ctx, bson.M{"role": "admin", "user_id": bson.M{"$ne": exclude}}, options.Find().SetLimit(100).SetProjection(bson.M{"user_id": 1}))
	if err != nil {
		return err
	}
	var admins []User
	err = cursor.All(ctx, &admins)
	cursor.Close(ctx)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(admins))
	for _, admin := range admins {
		ids = append(ids, admin.ID)
	}
	publish(RealtimeEvent{Type: eventType, Recipients: ids, AdminsOnly: true})
	return nil
}
