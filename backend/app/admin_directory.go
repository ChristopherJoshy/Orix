package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type adminPerson struct {
	Phone      *string   `json:"phone" bson:"phone"`
	Gender     *string   `json:"gender" bson:"gender"`
	ID         string    `json:"user_id" bson:"user_id"`
	Name       string    `json:"display_name" bson:"display_name"`
	Email      string    `json:"email" bson:"email"`
	Photo      *string   `json:"photo_url" bson:"photo_url"`
	Occupation string    `json:"occupation" bson:"occupation"`
	Role       string    `json:"role" bson:"role"`
	Status     string    `json:"status" bson:"status"`
	Created    time.Time `json:"-" bson:"created_at"`
}
type adminDirectoryRepository interface {
	AdminPeople(context.Context) ([]adminPerson, error)
	AdminGroups(context.Context) ([]adminDirectoryGroup, error)
	AdminConversation(context.Context, string, string) ([]ChatMessage, error)
	AdminSend(context.Context, string, string, string, string) (ChatMessage, error)
}
type adminDirectoryGroup struct {
	AdminGroupSummary `bson:",inline"`
	Created           time.Time `json:"-" bson:"created_at"`
}

func (m mongoUsers) AdminPeople(ctx context.Context) ([]adminPerson, error) {
	query := bson.M{}
	pageQuery(ctx, query, "created_at", "user_id")
	cur, err := m.collection.Find(ctx, query, options.Find().SetProjection(bson.M{"user_id": 1, "display_name": 1, "email": 1, "photo_url": 1, "occupation": 1, "phone": 1, "gender": 1, "role": 1, "status": 1, "created_at": 1}).SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "user_id", Value: -1}}).SetLimit(pageLimit(ctx, 50)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	rows := []adminPerson{}
	if err = cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	return trimPage(ctx, rows, func(p adminPerson) (time.Time, string) { return p.Created, p.ID }), nil
}
func (m mongoUsers) enrichMembers(ctx context.Context, rows []AdminGroupMember) error {
	ids := make([]string, 0, len(rows))
	for _, member := range rows {
		ids = append(ids, member.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	cur, err := m.collection.Find(ctx, bson.M{"user_id": bson.M{"$in": ids}}, options.Find().SetProjection(bson.M{"user_id": 1, "display_name": 1, "email": 1, "photo_url": 1, "occupation": 1, "phone": 1, "gender": 1, "status": 1}))
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	people := []adminPerson{}
	if err = cur.All(ctx, &people); err != nil {
		return err
	}
	byID := map[string]adminPerson{}
	for _, p := range people {
		byID[p.ID] = p
	}
	for i := range rows {
		p := byID[rows[i].ID]
		rows[i].Name = p.Name
		rows[i].Email = p.Email
		rows[i].Photo = p.Photo
		rows[i].Occupation = p.Occupation
		rows[i].Phone = p.Phone
		rows[i].Gender = p.Gender
		rows[i].Status = p.Status
	}
	return nil
}
func (m mongoUsers) AdminGroups(ctx context.Context) ([]adminDirectoryGroup, error) {
	query := bson.M{}
	pageQuery(ctx, query, "created_at", "group_id")
	cur, err := m.collection.Database().Collection("ride_groups").Find(ctx, query, options.Find().SetProjection(bson.M{"group_id": 1, "status": 1, "mode": 1, "pickup_summary": 1, "drop_summary": 1, "departure_start": 1, "departure_end": 1, "members": 1, "created_at": 1}).SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "group_id", Value: -1}}).SetLimit(pageLimit(ctx, 50)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	rows := []adminDirectoryGroup{}
	if err = cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	rows = trimPage(ctx, rows, func(g adminDirectoryGroup) (time.Time, string) { return g.Created, g.ID })
	allMembers := []AdminGroupMember{}
	for i := range rows {
		if rows[i].Members == nil {
			rows[i].Members = []AdminGroupMember{}
		}
		allMembers = append(allMembers, rows[i].Members...)
	}
	if err = m.enrichMembers(ctx, allMembers); err != nil {
		return nil, err
	}
	offset := 0
	for i := range rows {
		copy(rows[i].Members, allMembers[offset:offset+len(rows[i].Members)])
		offset += len(rows[i].Members)
	}
	return rows, nil
}
func (m mongoUsers) AdminConversation(ctx context.Context, actor, group string) ([]ChatMessage, error) {
	var existing bson.M
	db := m.collection.Database()
	if err := db.Collection("ride_groups").FindOne(ctx, bson.M{"group_id": group}, options.FindOne().SetProjection(bson.M{"_id": 1})).Decode(&existing); err != nil {
		return nil, err
	}
	query := bson.M{"group_id": group}
	pageQuery(ctx, query, "created_at", "message_id")
	cur, err := db.Collection("chat_messages").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "message_id", Value: -1}}).SetLimit(pageLimit(ctx, 50)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	rows := []ChatMessage{}
	if err = cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	rows = trimPage(ctx, rows, func(msg ChatMessage) (time.Time, string) { return msg.Created, msg.ID })
	for i := range rows {
		rows[i].Mine = rows[i].UserID == actor
		if !rows[i].Mine {
			rows[i].ClientID = ""
		}
	}
	if _, err = WriteAudit(ctx, db, bson.M{"actor_id": actor, "action": "viewed_group_chat", "target_id": group, "created_at": time.Now().UTC()}); err != nil {
		return nil, err
	}
	return rows, nil
}
func (m mongoUsers) AdminSend(ctx context.Context, actor, group, content, client string) (ChatMessage, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return ChatMessage{}, err
	}
	id := hex.EncodeToString(raw)
	key := actor + ":" + group + ":" + client
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return ChatMessage{}, err
	}
	defer session.EndSession(ctx)
	value, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		admin, err := m.authorizeAdminMutation(tx, actor)
		if err != nil {
			return nil, err
		}
		var g GroupRecord
		if err = db.Collection("ride_groups").FindOne(tx, bson.M{"group_id": group}).Decode(&g); err != nil {
			return nil, err
		}
		blocked, blockErr := m.BlockedPeers(tx, actor)
		if blockErr != nil {
			return nil, blockErr
		}
		for _, member := range g.Members {
			if blocked[member.UserID] {
				return nil, ErrGroupUnavailable
			}
		}
		var existing ChatMessage
		err = db.Collection("chat_messages").FindOne(tx, bson.M{"dedupe_key": key}).Decode(&existing)
		if err == nil {
			if existing.Content != content {
				return nil, ErrMessageConflict
			}
			return existing, nil
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}
		switch g.Status {
		case "pending", "confirming", "active", "in_progress":
		default:
			return nil, ErrGroupUnavailable
		}
		result, err := db.Collection("ride_groups").UpdateOne(tx, bson.M{"group_id": group, "status": g.Status}, bson.M{"$inc": bson.M{"chat_sequence": 1}})
		if err != nil {
			return nil, err
		}
		if result.MatchedCount != 1 {
			return nil, ErrGroupUnavailable
		}
		msg := ChatMessage{ID: id, GroupID: group, UserID: actor, Name: admin.Name + " (ORIX admin)", Type: "user", Content: content, Created: time.Now().UTC(), ClientID: client, DedupeKey: key}
		if _, err = db.Collection("chat_messages").InsertOne(tx, msg); err != nil {
			return nil, err
		}
		recipients := []string{}
		for _, member := range g.Members {
			if member.UserID != actor {
				recipients = append(recipients, member.UserID)
			}
		}
		if _, err = db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "new_chat_message", "group_id": group, "message_id": id, "user_id": actor, "recipient_ids": recipients, "status": "pending", "attempts": 0, "created_at": msg.Created}); err != nil {
			return nil, err
		}
		if _, err = WriteAudit(tx, db, bson.M{"actor_id": actor, "action": "sent_group_message", "target_id": group, "created_at": msg.Created}); err != nil {
			return nil, err
		}
		return msg, nil
	})
	if err != nil {
		return ChatMessage{}, err
	}
	msg, ok := value.(ChatMessage)
	if !ok {
		return ChatMessage{}, errors.New("missing message")
	}
	msg.Mine = true
	return msg, nil
}
func (a *AuthDependencies) adminDirectory(action string) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		repo, ok := a.Users.(adminDirectoryRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Admin directory unavailable"})
			return
		}
		group := r.PathValue("group_id")
		if (action == "chat" || action == "send") && (group == "" || len(group) > 100) {
			writeJSON(w, 422, map[string]string{"detail": "Invalid group"})
			return
		}
		if action == "send" {
			var input struct {
				Content string `json:"content"`
				Client  string `json:"client_message_id"`
			}
			if decodeBody(r, &input) != nil {
				writeJSON(w, 422, map[string]string{"detail": "Invalid message"})
				return
			}
			input.Content = strings.TrimSpace(input.Content)
			if input.Content == "" || utf8.RuneCountInString(input.Content) > 1000 || input.Client == "" || len(input.Client) > 100 || strings.ContainsAny(input.Client, ":\r\n") {
				writeJSON(w, 422, map[string]string{"detail": "Invalid message"})
				return
			}
			msg, err := repo.AdminSend(r.Context(), user.ID, group, input.Content, input.Client)
			if adminAccessDenied(w, err) {
				return
			}
			if err != nil {
				chatFailure(w, err)
				return
			}
			writeJSON(w, 200, msg)
			return
		}
		paged, valid := preparePage(w, r, "admin-directory:"+action+":"+group, 50)
		if !valid {
			return
		}
		r = paged
		var result any
		var err error
		switch action {
		case "people":
			result, err = repo.AdminPeople(r.Context())
		case "groups":
			result, err = repo.AdminGroups(r.Context())
		case "chat":
			result, err = repo.AdminConversation(r.Context(), user.ID, group)
		}
		if err != nil {
			chatFailure(w, err)
			return
		}
		writePageHeader(w, r)
		writeJSON(w, 200, result)
	}
}

func (a *AuthDependencies) adminAvatar(w http.ResponseWriter, r *http.Request, user User) {
	m, ok := a.Users.(mongoUsers)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Avatar storage unavailable"})
		return
	}
	var avatar Avatar
	err := m.collection.Database().Collection("user_avatars").FindOne(r.Context(), bson.M{"user_id": r.PathValue("user_id")}).Decode(&avatar)
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Avatar not found"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Avatar unavailable"})
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", avatar.ContentType)
	_, _ = w.Write(avatar.Data)
}
