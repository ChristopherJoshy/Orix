package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type ChatMessage struct {
	ID        string    `json:"message_id" bson:"message_id"`
	GroupID   string    `json:"-" bson:"group_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Name      string    `json:"user_display_name" bson:"user_display_name"`
	Type      string    `json:"message_type" bson:"message_type"`
	Content   string    `json:"content" bson:"content"`
	Created   time.Time `json:"created_at" bson:"created_at"`
	Mine      bool      `json:"is_mine" bson:"-"`
	ClientID  string    `json:"client_message_id,omitempty" bson:"client_message_id,omitempty"`
	DedupeKey string    `json:"-" bson:"dedupe_key"`
}
type ChatCursor struct {
	Created time.Time `json:"t"`
	ID      string    `json:"id"`
}
type ChatRepository interface {
	SendChat(context.Context, string, string, string, string) (ChatMessage, error)
	ReadChat(context.Context, string, string, int, *ChatCursor) ([]ChatMessage, error)
}

var ErrMessageConflict = errors.New("client message ID already belongs to different content")

func (m mongoUsers) SendChat(ctx context.Context, user, groupID, content, clientID string) (ChatMessage, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return ChatMessage{}, err
	}
	id := hex.EncodeToString(raw)
	if clientID == "" {
		clientID = id
	}
	key := user + ":" + groupID + ":" + clientID
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return ChatMessage{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		messages := db.Collection("chat_messages")
		var profile User
		if err := m.collection.FindOne(tx, bson.M{"user_id": user}).Decode(&profile); err != nil {
			return nil, err
		}
		if !profile.Onboarded || accountRestricted(profile) {
			return nil, ErrGroupUnavailable
		}
		var group GroupRecord
		if err := db.Collection("ride_groups").FindOne(tx, bson.M{"group_id": groupID, "members.user_id": user}).Decode(&group); err != nil {
			return nil, err
		}
		blocked, blockErr := m.BlockedPeers(tx, user)
		if blockErr != nil {
			return nil, blockErr
		}
		for _, member := range group.Members {
			if blocked[member.UserID] {
				return nil, ErrGroupUnavailable
			}
		}
		var existing ChatMessage
		lookup := messages.FindOne(tx, bson.M{"dedupe_key": key}).Decode(&existing)
		if lookup == nil {
			if existing.Content != content {
				return nil, ErrMessageConflict
			}
			existing.Mine = true
			return existing, nil
		}
		if !errors.Is(lookup, mongo.ErrNoDocuments) {
			return nil, lookup
		}
		switch group.Status {
		case "pending", "confirming", "active", "in_progress":
		default:
			return nil, ErrGroupUnavailable
		}
		// Touch the group to serialize sending against leaving/cancelling a group.
		if _, err := db.Collection("ride_groups").UpdateOne(tx, bson.M{"group_id": groupID, "members.user_id": user}, bson.M{"$inc": bson.M{"chat_sequence": 1}}); err != nil {
			return nil, err
		}
		message := ChatMessage{ID: id, GroupID: groupID, UserID: user, Name: profile.Name, Type: "user", Content: content, Created: time.Now().UTC(), Mine: true, ClientID: clientID, DedupeKey: key}
		if _, err := messages.InsertOne(tx, message); err != nil {
			return nil, err
		}
		recipients := []string{}
		for _, member := range group.Members {
			if member.UserID != user {
				recipients = append(recipients, member.UserID)
			}
		}
		_, err := db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "new_chat_message", "group_id": groupID, "message_id": id, "user_id": user, "recipient_ids": recipients, "status": "pending", "attempts": 0, "created_at": message.Created})
		if err != nil {
			return nil, err
		}
		return message, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return ChatMessage{}, err
	}
	message, ok := result.(ChatMessage)
	if !ok {
		return ChatMessage{}, errors.New("missing message result")
	}
	return message, nil
}
func (m mongoUsers) ReadChat(ctx context.Context, user, groupID string, limit int, before *ChatCursor) ([]ChatMessage, error) {
	// Read authorization and messages from a consistent snapshot.
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		var group GroupRecord
		if err := m.collection.Database().Collection("ride_groups").FindOne(tx, bson.M{"group_id": groupID, "members.user_id": user}).Decode(&group); err != nil {
			return nil, err
		}
		query := bson.M{"group_id": groupID}
		blocked, blockErr := m.BlockedPeers(tx, user)
		if blockErr != nil {
			return nil, blockErr
		}
		peers := []string{}
		for peer := range blocked {
			peers = append(peers, peer)
		}
		if len(peers) > 0 {
			query["user_id"] = bson.M{"$nin": peers}
		}
		if before != nil {
			query["$or"] = bson.A{bson.M{"created_at": bson.M{"$lt": before.Created}}, bson.M{"created_at": before.Created, "message_id": bson.M{"$lt": before.ID}}}
		}
		cursor, err := m.collection.Database().Collection("chat_messages").Find(tx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "message_id", Value: -1}}).SetLimit(int64(limit)))
		if err != nil {
			return nil, err
		}
		defer cursor.Close(tx)
		messages := []ChatMessage{}
		if err = cursor.All(tx, &messages); err != nil {
			return nil, err
		}
		for i := range messages {
			messages[i].Mine = messages[i].UserID == user
			if !messages[i].Mine {
				messages[i].ClientID = ""
			}
		}
		// Existing clients display a page chronologically, oldest to newest.
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
		return messages, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()))
	if err != nil {
		return nil, err
	}
	messages, ok := result.([]ChatMessage)
	if !ok {
		return nil, errors.New("missing message page")
	}
	return messages, nil
}
func (a *AuthDependencies) chatStorage(w http.ResponseWriter, user User) (ChatRepository, bool) {
	if _, ok := a.groupStorage(w, user); !ok {
		return nil, false
	}
	repository, ok := a.Users.(ChatRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Chat storage is not configured"})
	}
	return repository, ok
}
func chatFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Group not found"})
		return
	}
	if errors.Is(err, ErrGroupUnavailable) || errors.Is(err, ErrMessageConflict) {
		writeJSON(w, 409, map[string]string{"detail": "Message cannot be sent in the current group state"})
		return
	}
	writeJSON(w, 503, map[string]string{"detail": "Chat is temporarily unavailable"})
}
func (a *AuthDependencies) sendChat(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.chatStorage(w, user)
	if !ok {
		return
	}
	var input struct {
		Content  string `json:"content"`
		ClientID string `json:"client_message_id"`
	}
	if decodeBody(r, &input) != nil {
		writeJSON(w, 422, map[string]string{"detail": "Invalid chat fields"})
		return
	}
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" || utf8.RuneCountInString(input.Content) > 1000 || len(input.ClientID) > 100 || strings.ContainsAny(input.ClientID, ":\r\n") {
		writeJSON(w, 422, map[string]string{"detail": "Messages must contain 1–1000 characters and a valid client ID"})
		return
	}
	message, err := repository.SendChat(r.Context(), user.ID, r.PathValue("group_id"), input.Content, input.ClientID)
	if err != nil {
		chatFailure(w, err)
		return
	}
	writeJSON(w, 200, message)
}
func (a *AuthDependencies) readChat(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.chatStorage(w, user)
	if !ok {
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			writeJSON(w, 422, map[string]string{"detail": "Limit must be between 1 and 100"})
			return
		}
		limit = value
	}
	var before *ChatCursor
	if raw := r.URL.Query().Get("before"); raw != "" {
		if len(raw) > 512 {
			writeJSON(w, 422, map[string]string{"detail": "Invalid message cursor"})
			return
		}
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		var cursor ChatCursor
		if err != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.Created.IsZero() || cursor.ID == "" || len(cursor.ID) > 100 {
			writeJSON(w, 422, map[string]string{"detail": "Invalid message cursor"})
			return
		}
		before = &cursor
	}
	messages, err := repository.ReadChat(r.Context(), user.ID, r.PathValue("group_id"), limit, before)
	if err != nil {
		chatFailure(w, err)
		return
	}
	if len(messages) == limit {
		data, _ := json.Marshal(ChatCursor{messages[0].Created, messages[0].ID})
		w.Header().Set("X-Next-Cursor", base64.RawURLEncoding.EncodeToString(data))
	}
	writeJSON(w, 200, messages)
}
