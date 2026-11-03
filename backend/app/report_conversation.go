package server

import (
	"context"
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

type ReportMessage struct {
	ID      string    `json:"message_id" bson:"message_id"`
	Sender  string    `json:"-" bson:"sender_id"`
	Admin   bool      `json:"is_admin" bson:"is_admin"`
	Content string    `json:"content" bson:"content"`
	Created time.Time `json:"created_at" bson:"created_at"`
}
type ReportConversation struct {
	Report   ReportRecord    `json:"report"`
	Messages []ReportMessage `json:"messages"`
}
type ReportConversationRepository interface {
	ReportConversation(context.Context, User, string, bool, string, string) (ReportConversation, error)
}

// Only the reporter and an uninvolved authorized moderator can read or reply.
// The accused is never a participant in the report conversation.
func reportParticipant(report ReportRecord, user User, admin bool) bool {
	if admin {
		return user.Role == "admin" && user.ID != report.Reporter && user.ID != report.Accused
	}
	return user.ID == report.Reporter
}
func (m mongoUsers) ReportConversation(ctx context.Context, user User, id string, admin bool, clientID, content string) (ReportConversation, error) {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return ReportConversation{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if admin {
			if _, err := m.authorizeAdminMutation(tx, user.ID); err != nil {
				return nil, err
			}
		}
		var stored struct {
			ReportRecord `bson:",inline"`
			Messages     []ReportMessage `bson:"messages"`
		}
		collection := m.collection.Database().Collection("reports")
		if err := collection.FindOne(tx, bson.M{"report_id": id}).Decode(&stored); err != nil {
			return nil, err
		}
		if !reportParticipant(stored.ReportRecord, user, admin) {
			return nil, mongo.ErrNoDocuments
		}
		messages := stored.Messages
		if messages == nil {
			messages = []ReportMessage{}
		}
		// Existing reports retain their original note as the first message.
		if len(messages) == 0 && strings.TrimSpace(stored.Note) != "" {
			messages = append(messages, ReportMessage{ID: id + "-initial", Sender: stored.Reporter, Content: stored.Note, Created: stored.Created})
		}
		if content != "" {
			for _, message := range messages {
				if message.ID == clientID && message.Sender == user.ID {
					if message.Content != content {
						return nil, ErrModerationConflict
					}
					return ReportConversation{stored.ReportRecord, messages}, nil
				}
			}
			if stored.Status != "pending" {
				return nil, ErrModerationConflict
			}
			if len(messages) >= 200 {
				return nil, ErrTicketLimit
			}
			// Server derives identity and role. A per-report write serializes replies
			// against other replies and moderator resolution in the same transaction.
			messages = append(messages, ReportMessage{ID: clientID, Sender: user.ID, Admin: admin, Content: content, Created: time.Now().UTC()})
			if _, err := collection.UpdateOne(tx, bson.M{"report_id": id, "status": "pending"}, bson.M{"$set": bson.M{"messages": messages, "updated_at": time.Now().UTC()}, "$inc": bson.M{"conversation_version": 1}}); err != nil {
				return nil, err
			}
			if !admin {
				if _, err := m.collection.Database().Collection("notification_outbox").InsertOne(tx, bson.M{"type": "report_followup", "report_id": id, "status": "pending", "attempts": 0, "created_at": time.Now().UTC()}); err != nil {
					return nil, err
				}
			}
			if admin {
				// The reply and its notification commit together; retries return above.
				_, err := m.collection.Database().Collection("notification_outbox").InsertOne(tx, bson.M{
					"_id":  stableID("report-reply:" + id + ":" + user.ID + ":" + clientID),
					"type": "report_reply", "report_id": id, "target_user_id": stored.Reporter,
					"status": "pending", "attempts": 0, "created_at": time.Now().UTC(),
				})
				if err != nil {
					return nil, err
				}
			}
		}
		return ReportConversation{stored.ReportRecord, messages}, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return ReportConversation{}, err
	}
	return result.(ReportConversation), nil
}
func (a *AuthDependencies) reportConversation(admin bool) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		repository, ok := a.Users.(ReportConversationRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Report conversations unavailable"})
			return
		}
		var input struct {
			ID      string `json:"client_message_id"`
			Content string `json:"content"`
		}
		if r.Method == http.MethodPost {
			if decodeBody(r, &input) != nil {
				writeJSON(w, 422, map[string]string{"detail": "Provide a message and client message ID"})
				return
			}
			input.Content = strings.TrimSpace(input.Content)
			input.ID = strings.TrimSpace(input.ID)
			if len(input.ID) < 8 || len(input.ID) > 100 || input.Content == "" || utf8.RuneCountInString(input.Content) > 2000 {
				writeJSON(w, 422, map[string]string{"detail": "Message must contain 1–2000 characters and a valid client ID"})
				return
			}
		}
		conversation, err := repository.ReportConversation(r.Context(), user, r.PathValue("report_id"), admin, input.ID, input.Content)
		if err != nil {
			reportFailure(w, err)
			return
		}
		writeJSON(w, 200, conversation)
	}
}
