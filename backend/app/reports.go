package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
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

type ReportInput struct {
	Accused  string  `json:"accused_user_id"`
	GroupID  *string `json:"group_id"`
	Note     string  `json:"note"`
	Category string  `json:"category"`
}
type ReportRecord struct {
	ID             string        `json:"report_id" bson:"report_id"`
	Reporter       string        `json:"-" bson:"reporter_user_id"`
	Accused        string        `json:"accused_user_id" bson:"accused_user_id"`
	GroupID        *string       `json:"group_id" bson:"group_id"`
	Note           string        `json:"note" bson:"note"`
	Category       string        `json:"category" bson:"category"`
	Status         string        `json:"status" bson:"status"`
	Created        time.Time     `json:"created_at" bson:"created_at"`
	Resolution     string        `json:"resolution,omitempty" bson:"resolution"`
	ResolutionNote string        `json:"-" bson:"resolution_note"`
	ResolvedBy     string        `json:"-" bson:"resolved_by"`
	ResolvedAt     *time.Time    `json:"resolved_at" bson:"resolved_at"`
	Evidence       []ChatMessage `json:"-" bson:"chat_evidence"`
}
type ResolutionInput struct {
	Action string `json:"resolution"`
	Note   string `json:"note"`
	Hours  int    `json:"suspend_hours"`
}
type ReportRepository interface {
	CreateReport(context.Context, string, ReportInput) (string, error)
	Reports(context.Context, string, bool) ([]ReportRecord, error)
	ResolveReport(context.Context, string, string, ResolutionInput) error
}

var ErrModerationConflict = errors.New("moderation conflict")

func (m mongoUsers) CreateReport(ctx context.Context, reporter string, input ReportInput) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	id := hex.EncodeToString(raw)
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return "", err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		var accused User
		if err := m.collection.FindOne(tx, bson.M{"user_id": input.Accused}).Decode(&accused); err != nil {
			return nil, err
		}
		record := ReportRecord{ID: id, Reporter: reporter, Accused: input.Accused, GroupID: input.GroupID, Note: input.Note, Category: input.Category, Status: "pending", Created: time.Now().UTC()}
		if input.GroupID != nil {
			var group GroupRecord
			if err := db.Collection("ride_groups").FindOne(tx, bson.M{"group_id": *input.GroupID, "members.user_id": bson.M{"$all": []string{reporter, input.Accused}}}).Decode(&group); err != nil {
				return nil, err
			}
			cursor, err := db.Collection("chat_messages").Find(tx, bson.M{"group_id": *input.GroupID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "message_id", Value: -1}}).SetLimit(20))
			if err != nil {
				return nil, err
			}
			defer cursor.Close(tx)
			if err = cursor.All(tx, &record.Evidence); err != nil {
				return nil, err
			}
		}
		_, err := db.Collection("reports").InsertOne(tx, record)
		if err != nil {
			return nil, err
		}
		if _, err = db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "report_created", "report_id": id, "status": "pending", "attempts": 0, "created_at": record.Created}); err != nil {
			return nil, err
		}
		_, err = WriteAudit(tx, db, bson.M{"actor_id": reporter, "action": "created_report", "target_id": id, "created_at": record.Created})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return id, err
}
func (m mongoUsers) Reports(ctx context.Context, user string, admin bool) ([]ReportRecord, error) {
	query := bson.M{"reporter_user_id": user}
	if admin {
		query = bson.M{"status": "pending", "reporter_user_id": bson.M{"$ne": user}, "accused_user_id": bson.M{"$ne": user}}
	}
	pageQuery(ctx, query, "created_at", "report_id")
	cursor, err := m.collection.Database().Collection("reports").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "report_id", Value: -1}}).SetLimit(pageLimit(ctx, 100)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	records := []ReportRecord{}
	err = cursor.All(ctx, &records)
	if err == nil {
		records = trimPage(ctx, records, func(report ReportRecord) (time.Time, string) { return report.Created, report.ID })
	}
	return records, err
}
func (m mongoUsers) ResolveReport(ctx context.Context, admin, id string, input ResolutionInput) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		if _, err := m.authorizeAdminMutation(tx, admin); err != nil {
			return nil, err
		}
		var report ReportRecord
		if err := db.Collection("reports").FindOne(tx, bson.M{"report_id": id}).Decode(&report); err != nil {
			return nil, err
		}
		if report.Reporter == admin || report.Accused == admin {
			return nil, ErrModerationConflict
		}
		if report.Status != "pending" {
			if report.Resolution == input.Action && report.ResolvedBy == admin {
				return nil, nil
			}
			return nil, ErrModerationConflict
		}
		now := time.Now().UTC()
		status := "resolved"
		if input.Action == "dismiss" {
			status = "dismissed"
		}
		if input.Action == "ban" || input.Action == "suspend" {
			var target User
			if err := m.collection.FindOne(tx, bson.M{"user_id": report.Accused}).Decode(&target); err != nil {
				return nil, err
			}
			if target.Role == "admin" {
				return nil, ErrModerationConflict
			}
			restriction := "banned"
			changes := bson.M{"status": restriction, "ban_reason": input.Note, "updated_at": now}
			if input.Action == "suspend" {
				restriction = "suspended"
				changes["status"] = restriction
				changes["suspension_expires_at"] = now.Add(time.Duration(input.Hours) * time.Hour)
			} else {
				changes["suspension_expires_at"] = nil
			}
			if _, err := m.collection.UpdateOne(tx, bson.M{"user_id": report.Accused}, bson.M{"$set": changes}); err != nil {
				return nil, err
			}
			cursor, err := db.Collection("ride_groups").Find(tx, bson.M{"members.user_id": report.Accused, "status": bson.M{"$in": []string{"pending", "confirming", "active", "in_progress"}}})
			if err != nil {
				return nil, err
			}
			defer cursor.Close(tx)
			var groups []GroupRecord
			if err = cursor.All(tx, &groups); err != nil {
				return nil, err
			}
			for _, group := range groups {
				if _, err = db.Collection("ride_groups").UpdateOne(tx, bson.M{"group_id": group.ID}, bson.M{"$set": bson.M{"status": "cancelled", "updated_at": now}}); err != nil {
					return nil, err
				}
				if _, err = m.rides().UpdateMany(tx, bson.M{"group_id": group.ID, "status": "matched"}, bson.M{"$set": bson.M{"status": "cancelled"}}); err != nil {
					return nil, err
				}
				if _, err = db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "group_cancelled", "group_id": group.ID, "status": "pending", "attempts": 0, "created_at": now}); err != nil {
					return nil, err
				}
			}
			if _, err = m.rides().UpdateMany(tx, bson.M{"user_id": report.Accused, "status": bson.M{"$in": []string{"pending", "matching"}}}, bson.M{"$set": bson.M{"status": "cancelled"}}); err != nil {
				return nil, err
			}
		}
		if _, err := db.Collection("reports").UpdateOne(tx, bson.M{"report_id": id, "status": "pending"}, bson.M{"$set": bson.M{"status": status, "resolution": input.Action, "resolution_note": input.Note, "resolved_by": admin, "resolved_at": now}}); err != nil {
			return nil, err
		}
		if _, err := WriteAudit(tx, db, bson.M{"actor_id": admin, "action": "resolved_report", "target_id": id, "resolution": input.Action, "note": input.Note, "created_at": now}); err != nil {
			return nil, err
		}
		if input.Action == "warn" {
			if _, err := db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "moderation_warning", "target_user_id": report.Accused, "status": "pending", "attempts": 0, "created_at": now}); err != nil {
				return nil, err
			}
		}
		_, err := db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "report_resolved", "report_id": report.ID, "target_user_id": report.Reporter, "status": "pending", "attempts": 0, "created_at": now})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}
func (a *AuthDependencies) admin(secret string, next func(http.ResponseWriter, *http.Request, User)) http.HandlerFunc {
	return a.authenticated(func(w http.ResponseWriter, r *http.Request, user User) {
		supplied := r.Header.Get("X-Admin-Secret")
		if secret == "" || user.Role != "admin" || subtle.ConstantTimeCompare([]byte(supplied), []byte(secret)) != 1 {
			writeJSON(w, 403, map[string]string{"detail": "Administrator access required"})
			return
		}
		next(w, r, user)
	})
}
func (a *AuthDependencies) reportHandler(action string) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		if !user.Onboarded {
			writeJSON(w, 403, map[string]string{"detail": "Complete onboarding first"})
			return
		}
		repository, ok := a.Users.(ReportRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Report storage unavailable"})
			return
		}
		switch action {
		case "create":
			var input ReportInput
			if decodeBody(r, &input) != nil || input.Accused == "" || input.Accused == user.ID || len(input.Accused) > 100 || utf8.RuneCountInString(input.Note) > 1000 || input.GroupID != nil && (*input.GroupID == "" || len(*input.GroupID) > 100) {
				writeJSON(w, 422, map[string]string{"detail": "Provide a valid user and report context"})
				return
			}
			input.Note = strings.TrimSpace(input.Note)
			if input.Category == "" {
				input.Category = "other"
			}
			switch input.Category {
			case "harassment", "unsafe_driving", "no_show", "spam", "other":
			default:
				writeJSON(w, 422, map[string]string{"detail": "Invalid report category"})
				return
			}
			id, err := repository.CreateReport(r.Context(), user.ID, input)
			if err != nil {
				reportFailure(w, err)
				return
			}
			writeJSON(w, 201, map[string]string{"report_id": id, "message": "Your report has been submitted for review"})
		case "own", "admin":
			paged, valid := preparePage(w, r, "reports:"+action+":"+user.ID, 100)
			if !valid {
				return
			}
			r = paged
			reports, err := repository.Reports(r.Context(), user.ID, action == "admin")
			if err != nil {
				reportFailure(w, err)
				return
			}
			writePageHeader(w, r)
			if action == "admin" {
				rows := []map[string]any{}
				for _, report := range reports {
					rows = append(rows, map[string]any{"report": report, "reporter_user_id": report.Reporter, "chat_evidence": report.Evidence})
				}
				writeJSON(w, 200, map[string]any{"reports": rows})
				return
			}
			writeJSON(w, 200, map[string]any{"reports": reports})
		case "resolve":
			var input ResolutionInput
			if decodeBody(r, &input) != nil || strings.TrimSpace(input.Note) == "" || utf8.RuneCountInString(input.Note) > 1000 {
				writeJSON(w, 422, map[string]string{"detail": "A resolution and review note are required"})
				return
			}
			switch input.Action {
			case "warn", "ban", "dismiss":
			case "suspend":
				if input.Hours < 1 || input.Hours > 720 {
					writeJSON(w, 422, map[string]string{"detail": "Suspension must last 1–720 hours"})
					return
				}
			default:
				writeJSON(w, 422, map[string]string{"detail": "Invalid resolution"})
				return
			}
			if err := repository.ResolveReport(r.Context(), user.ID, r.PathValue("report_id"), input); err != nil {
				reportFailure(w, err)
				return
			}
			writeJSON(w, 200, map[string]string{"message": "Report reviewed"})
		}
	}
}
func reportFailure(w http.ResponseWriter, err error) {
	if adminAccessDenied(w, err) {
		return
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "User, group or report not found"})
		return
	}
	if errors.Is(err, ErrTicketLimit) {
		writeJSON(w, 409, map[string]string{"detail": "This conversation has reached its message limit"})
		return
	}
	if errors.Is(err, ErrModerationConflict) {
		writeJSON(w, 409, map[string]string{"detail": "This review needs a different moderator or has already been resolved"})
		return
	}
	writeJSON(w, 503, map[string]string{"detail": "Unable to process report. Please try again."})
}
