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
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type TicketMessage struct {
	Sender    string    `json:"sender_id" bson:"sender_id"`
	Name      string    `json:"sender_name" bson:"sender_name"`
	Content   string    `json:"content" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Admin     bool      `json:"is_admin" bson:"is_admin"`
	RetryKey  string    `json:"-" bson:"retry_key,omitempty"`
}
type Ticket struct {
	ID           string          `json:"ticket_id" bson:"ticket_id"`
	UserID       string          `json:"user_id" bson:"user_id"`
	Type         string          `json:"type" bson:"type"`
	Subject      string          `json:"subject" bson:"subject"`
	Description  string          `json:"description" bson:"description"`
	Status       string          `json:"status" bson:"status"`
	Priority     string          `json:"priority" bson:"priority"`
	ReportedUser *string         `json:"reported_user_id" bson:"reported_user_id"`
	Messages     []TicketMessage `json:"messages" bson:"messages"`
	Created      time.Time       `json:"created_at" bson:"created_at"`
	Updated      time.Time       `json:"updated_at" bson:"updated_at"`
	Resolved     *time.Time      `json:"resolved_at" bson:"resolved_at"`
	ResolvedBy   *string         `json:"-" bson:"resolved_by"`
}
type TicketInput struct {
	Type         string  `json:"type"`
	Subject      string  `json:"subject"`
	Description  string  `json:"description"`
	Priority     string  `json:"priority"`
	ReportedUser *string `json:"reported_user_id"`
}
type SupportRepository interface {
	CreateTicket(context.Context, User, TicketInput) (Ticket, error)
	Tickets(context.Context, string, string, bool) ([]Ticket, error)
	ChangeTicket(context.Context, User, string, string, string, string, bool) (Ticket, error)
}

var ErrTicketLimit = errors.New("ticket or conversation limit reached")

func (m mongoUsers) CreateTicket(ctx context.Context, user User, input TicketInput) (Ticket, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return Ticket{}, err
	}
	id := "TICKET-" + hex.EncodeToString(raw)
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return Ticket{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		now := time.Now().UTC()
		// Serialize concurrent creates by the same user before checking ticket quotas.
		if _, err := m.collection.UpdateOne(tx, bson.M{"user_id": user.ID}, bson.M{"$inc": bson.M{"support_version": 1}}); err != nil {
			return nil, err
		}
		var current User
		if err := m.collection.FindOne(tx, bson.M{"user_id": user.ID}).Decode(&current); err != nil {
			return nil, err
		}
		if current.Status == "deleting" || current.Status == "deleted" || accountRestricted(current) && input.Type != "appeal" {
			return nil, ErrModerationConflict
		}
		open, err := db.Collection("support_tickets").CountDocuments(tx, bson.M{"user_id": user.ID, "status": bson.M{"$in": []string{"open", "in_progress"}}})
		if err != nil {
			return nil, err
		}
		daily, err := db.Collection("support_tickets").CountDocuments(tx, bson.M{"user_id": user.ID, "created_at": bson.M{"$gte": now.Truncate(24 * time.Hour)}})
		if err != nil {
			return nil, err
		}
		if open >= 5 || daily >= 10 {
			return nil, ErrTicketLimit
		}
		if input.ReportedUser != nil {
			if err := m.collection.FindOne(tx, bson.M{"user_id": *input.ReportedUser}).Err(); err != nil {
				return nil, err
			}
		}
		ticket := Ticket{ID: id, UserID: user.ID, Type: input.Type, Subject: input.Subject, Description: input.Description, Status: "open", Priority: "normal", ReportedUser: input.ReportedUser, Created: now, Updated: now, Messages: []TicketMessage{{Sender: user.ID, Name: user.Name, Content: input.Description, Timestamp: now, Admin: false}}}
		if _, err = db.Collection("support_tickets").InsertOne(tx, ticket); err != nil {
			return nil, err
		}
		_, err = WriteAudit(tx, db, bson.M{"actor_id": user.ID, "action": "created_ticket", "target_id": id, "created_at": now})
		return ticket, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return Ticket{}, err
	}
	ticket, ok := result.(Ticket)
	if !ok {
		return Ticket{}, errors.New("missing ticket result")
	}
	return ticket, nil
}
func (m mongoUsers) Tickets(ctx context.Context, user, id string, admin bool) ([]Ticket, error) {
	query := bson.M{"user_id": user}
	if admin {
		query = bson.M{"status": bson.M{"$in": []string{"open", "in_progress"}}}
	}
	if id != "" {
		query["ticket_id"] = id
		if admin {
			delete(query, "status")
		}
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}, {Key: "ticket_id", Value: -1}}).SetLimit(pageLimit(ctx, 100))
	if id == "" {
		pageQuery(ctx, query, "updated_at", "ticket_id")
		findOptions.SetProjection(bson.M{"messages": 0})
	}
	cursor, err := m.collection.Database().Collection("support_tickets").Find(ctx, query, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	records := []Ticket{}
	err = cursor.All(ctx, &records)
	if id == "" {
		records = trimPage(ctx, records, func(ticket Ticket) (time.Time, string) { return ticket.Updated, ticket.ID })
		for i := range records {
			records[i].Messages = []TicketMessage{}
		}
	}
	return records, err
}
func (m mongoUsers) ChangeTicket(ctx context.Context, actor User, id, action, content, retryID string, admin bool) (Ticket, error) {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return Ticket{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		var current User
		if err := m.collection.FindOne(tx, bson.M{"user_id": actor.ID}).Decode(&current); err != nil {
			return nil, err
		}
		if current.Status == "deleting" || current.Status == "deleted" || admin && (accountRestricted(current) || current.Role != "admin") {
			return nil, ErrModerationConflict
		}
		if admin {
			if _, err := m.authorizeAdminMutation(tx, actor.ID); err != nil {
				return nil, err
			}
		}
		query := bson.M{"ticket_id": id}
		if !admin {
			query["user_id"] = actor.ID
		}
		var ticket Ticket
		if err := db.Collection("support_tickets").FindOne(tx, query).Decode(&ticket); err != nil {
			return nil, err
		}
		if !admin && accountRestricted(current) && ticket.Type != "appeal" {
			return nil, ErrModerationConflict
		}
		now := time.Now().UTC()
		switch action {
		case "message":
			retryKey := ""
			if retryID != "" {
				retryKey = actor.ID + ":" + retryID
				for _, existing := range ticket.Messages {
					if existing.RetryKey == retryKey {
						if existing.Content != content {
							return nil, ErrModerationConflict
						}
						return ticket, nil
					}
				}
			}
			if ticket.Status == "closed" || ticket.Status == "resolved" {
				return nil, ErrGroupUnavailable
			}
			if len(ticket.Messages) >= 200 {
				return nil, ErrTicketLimit
			}
			sender, name := actor.ID, current.Name
			if admin {
				sender = "support"
				name = "ORIX Support"
				ticket.Status = "in_progress"
			}
			ticket.Messages = append(ticket.Messages, TicketMessage{Sender: sender, Name: name, Content: content, Timestamp: now, Admin: admin, RetryKey: retryKey})
		case "close", "resolved", "in_progress":
			if !admin && action != "close" {
				return nil, ErrModerationConflict
			}
			state := action
			if state == "close" {
				state = "closed"
			}
			if ticket.Status == state {
				return ticket, nil
			}
			ticket.Status = state
			if state == "closed" || state == "resolved" {
				ticket.Resolved = &now
				ticket.ResolvedBy = &actor.ID
			} else {
				ticket.Resolved = nil
				ticket.ResolvedBy = nil
			}
		case "reopen":
			if ticket.Status != "resolved" && ticket.Status != "closed" {
				return ticket, nil
			}
			if !admin {
				if _, err := m.collection.UpdateOne(tx, bson.M{"user_id": actor.ID}, bson.M{"$inc": bson.M{"support_version": 1}}); err != nil {
					return nil, err
				}
				open, err := db.Collection("support_tickets").CountDocuments(tx, bson.M{"user_id": actor.ID, "status": bson.M{"$in": []string{"open", "in_progress"}}})
				if err != nil {
					return nil, err
				}
				if open >= 5 {
					return nil, ErrTicketLimit
				}
			}
			ticket.Status = "open"
			ticket.Resolved = nil
			ticket.ResolvedBy = nil
		default:
			return nil, ErrModerationConflict
		}
		ticket.Updated = now
		if _, err := db.Collection("support_tickets").ReplaceOne(tx, query, ticket); err != nil {
			return nil, err
		}
		if _, err := WriteAudit(tx, db, bson.M{"actor_id": actor.ID, "action": "ticket_" + action, "target_id": id, "is_admin": admin, "created_at": now}); err != nil {
			return nil, err
		}
		if admin && actor.ID != ticket.UserID {
			if _, err := db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "support_reply", "target_user_id": ticket.UserID, "ticket_id": id, "status": "pending", "attempts": 0, "created_at": now}); err != nil {
				return nil, err
			}
		}
		return ticket, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return Ticket{}, err
	}
	ticket, ok := result.(Ticket)
	if !ok {
		return Ticket{}, errors.New("missing ticket result")
	}
	return ticket, nil
}
func (a *AuthDependencies) support(action string, admin bool) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		repository, ok := a.Users.(SupportRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Support storage unavailable"})
			return
		}
		if action == "list" {
			if r.PathValue("ticket_id") == "" {
				scope := "tickets:" + user.ID
				if admin {
					scope = "admin_" + scope
				}
				paged, valid := preparePage(w, r, scope, 100)
				if !valid {
					return
				}
				r = paged
			}
			records, err := repository.Tickets(r.Context(), user.ID, r.PathValue("ticket_id"), admin)
			if err != nil {
				supportFailure(w, err)
				return
			}
			if r.PathValue("ticket_id") != "" {
				if len(records) == 0 {
					writeJSON(w, 404, map[string]string{"detail": "Ticket not found"})
					return
				}
				writeJSON(w, 200, records[0])
				return
			}
			writePageHeader(w, r)
			writeJSON(w, 200, records)
			return
		}
		if action == "create" {
			var input TicketInput
			if decodeBody(r, &input) != nil {
				writeJSON(w, 422, map[string]string{"detail": "Invalid ticket fields"})
				return
			}
			input.Subject = strings.TrimSpace(input.Subject)
			input.Description = strings.TrimSpace(input.Description)
			if input.Subject == "" || utf8.RuneCountInString(input.Subject) > 150 || input.Description == "" || utf8.RuneCountInString(input.Description) > 4000 || input.ReportedUser != nil && (*input.ReportedUser == "" || len(*input.ReportedUser) > 100) {
				writeJSON(w, 422, map[string]string{"detail": "Provide a subject and description within the length limits"})
				return
			}
			switch input.Type {
			case "issue", "feature", "report_user", "other", "appeal":
			default:
				writeJSON(w, 422, map[string]string{"detail": "Invalid ticket type"})
				return
			}
			ticket, err := repository.CreateTicket(r.Context(), user, input)
			if err != nil {
				supportFailure(w, err)
				return
			}
			writeJSON(w, 201, ticket)
			return
		}
		content := ""
		retryID := ""
		if action == "message" {
			var input struct {
				Content string `json:"content"`
				RetryID string `json:"client_message_id"`
			}
			if decodeBody(r, &input) != nil {
				writeJSON(w, 422, map[string]string{"detail": "Invalid message"})
				return
			}
			retryID = input.RetryID
			if len(retryID) > 100 || strings.ContainsAny(retryID, " \t\r\n:") {
				writeJSON(w, 422, map[string]string{"detail": "Invalid message identifier"})
				return
			}
			content = strings.TrimSpace(input.Content)
			if content == "" || utf8.RuneCountInString(content) > 4000 {
				writeJSON(w, 422, map[string]string{"detail": "Messages must contain 1–4000 characters"})
				return
			}
		}
		ticket, err := repository.ChangeTicket(r.Context(), user, r.PathValue("ticket_id"), action, content, retryID, admin)
		if err != nil {
			supportFailure(w, err)
			return
		}
		writeJSON(w, 200, ticket)
	}
}
func supportFailure(w http.ResponseWriter, err error) {
	if adminAccessDenied(w, err) {
		return
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Ticket or user not found"})
		return
	}
	if errors.Is(err, ErrTicketLimit) {
		writeJSON(w, 429, map[string]string{"detail": "Ticket or conversation limit reached. Continue in a new ticket when available."})
		return
	}
	if errors.Is(err, ErrGroupUnavailable) || errors.Is(err, ErrModerationConflict) {
		writeJSON(w, 409, map[string]string{"detail": "This ticket action is not available"})
		return
	}
	writeJSON(w, 503, map[string]string{"detail": "Support is temporarily unavailable"})
}
