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

type RatingInput struct {
	Target  string  `json:"rated_user_id"`
	GroupID string  `json:"group_id"`
	Stars   int     `json:"rating"`
	Comment *string `json:"comment"`
}
type RatingRecord struct {
	ID      string    `json:"rating_id" bson:"rating_id"`
	Rater   string    `json:"-" bson:"rater_user_id"`
	Target  string    `json:"rated_user_id" bson:"rated_user_id"`
	GroupID string    `json:"-" bson:"group_id"`
	Stars   int       `json:"rating" bson:"rating"`
	Comment *string   `json:"-" bson:"comment"`
	Created time.Time `json:"created_at" bson:"created_at"`
}
type PendingRatingUser struct {
	ID    string  `json:"user_id"`
	Name  string  `json:"display_name"`
	Photo *string `json:"photo_url"`
}
type PendingRatingGroup struct {
	ID      string
	Users   []PendingRatingUser
	Members int
}
type RatingRepository interface {
	SubmitRating(context.Context, string, RatingInput) (RatingRecord, error)
	PendingRatings(context.Context, string, string) ([]PendingRatingGroup, error)
	AverageRating(context.Context, string, string) (*float64, error)
}

var ErrRatingConflict = errors.New("rating is not available or already differs")

func ratingAllowed(group GroupRecord, rater, target string) bool {
	if rater == target {
		return false
	}
	raterFound, targetFound := false, false
	for _, member := range group.Members {
		if member.UserID == rater {
			raterFound = group.Status == "completed" || member.Finished
		}
		if member.UserID == target {
			targetFound = group.Status == "completed" || member.Finished
		}
	}
	return (group.Status == "completed" || group.Status == "in_progress") && raterFound && targetFound
}
func sameRating(record RatingRecord, input RatingInput) bool {
	old, new := "", ""
	if record.Comment != nil {
		old = *record.Comment
	}
	if input.Comment != nil {
		new = *input.Comment
	}
	return record.Stars == input.Stars && old == new
}
func (m mongoUsers) SubmitRating(ctx context.Context, user string, input RatingInput) (RatingRecord, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return RatingRecord{}, err
	}
	record := RatingRecord{ID: hex.EncodeToString(raw), Rater: user, Target: input.Target, GroupID: input.GroupID, Stars: input.Stars, Comment: input.Comment, Created: time.Now().UTC()}
	filter := bson.M{"group_id": input.GroupID, "rater_user_id": user, "rated_user_id": input.Target}
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return RatingRecord{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		var group GroupRecord
		if err := db.Collection("ride_groups").FindOne(tx, bson.M{"group_id": input.GroupID, "members.user_id": user}).Decode(&group); err != nil {
			return nil, err
		}
		if !ratingAllowed(group, user, input.Target) {
			return nil, ErrRatingConflict
		}
		var existing RatingRecord
		lookup := db.Collection("ratings").FindOne(tx, filter).Decode(&existing)
		if lookup == nil {
			if !sameRating(existing, input) {
				return nil, ErrRatingConflict
			}
			return existing, nil
		}
		if !errors.Is(lookup, mongo.ErrNoDocuments) {
			return nil, lookup
		}
		if _, err := db.Collection("ratings").InsertOne(tx, record); err != nil {
			return nil, err
		}
		return record, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if mongo.IsDuplicateKeyError(err) {
		var existing RatingRecord
		if lookup := m.collection.Database().Collection("ratings").FindOne(ctx, filter).Decode(&existing); lookup == nil && sameRating(existing, input) {
			return existing, nil
		}
		return RatingRecord{}, ErrRatingConflict
	}
	if err != nil {
		return RatingRecord{}, err
	}
	saved, ok := result.(RatingRecord)
	if !ok {
		return RatingRecord{}, errors.New("missing rating result")
	}
	return saved, nil
}
func (m mongoUsers) PendingRatings(ctx context.Context, user, id string) ([]PendingRatingGroup, error) {
	db := m.collection.Database()
	query := bson.M{"members.user_id": user, "status": bson.M{"$in": []string{"completed", "in_progress"}}}
	if id != "" {
		query["group_id"] = id
	}
	cursor, err := db.Collection("ride_groups").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(100))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var groups []GroupRecord
	if err = cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	ids := []string{}
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	ratingsCursor, err := db.Collection("ratings").Find(ctx, bson.M{"rater_user_id": user, "group_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer ratingsCursor.Close(ctx)
	var ratings []RatingRecord
	if err = ratingsCursor.All(ctx, &ratings); err != nil {
		return nil, err
	}
	rated := map[string]bool{}
	for _, rating := range ratings {
		rated[rating.GroupID+":"+rating.Target] = true
	}
	pending := []PendingRatingGroup{}
	profileIDs := []string{}
	for _, group := range groups {
		row := PendingRatingGroup{ID: group.ID, Members: len(group.Members), Users: []PendingRatingUser{}}
		for _, member := range group.Members {
			if ratingAllowed(group, user, member.UserID) && !rated[group.ID+":"+member.UserID] {
				row.Users = append(row.Users, PendingRatingUser{ID: member.UserID})
				profileIDs = append(profileIDs, member.UserID)
			}
		}
		if len(row.Users) > 0 || id != "" {
			pending = append(pending, row)
		}
	}
	if id != "" && len(groups) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	profiles := map[string]User{}
	if len(profileIDs) > 0 {
		users, err := m.collection.Find(ctx, bson.M{"user_id": bson.M{"$in": profileIDs}})
		if err != nil {
			return nil, err
		}
		defer users.Close(ctx)
		var records []User
		if err = users.All(ctx, &records); err != nil {
			return nil, err
		}
		for _, profile := range records {
			profiles[profile.ID] = profile
		}
	}
	for i := range pending {
		for j := range pending[i].Users {
			profile, found := profiles[pending[i].Users[j].ID]
			if found {
				pending[i].Users[j].Name = profile.Name
				pending[i].Users[j].Photo = profile.Photo
			} else {
				pending[i].Users[j].Name = "Former traveler"
			}
		}
	}
	return pending, nil
}
func (m mongoUsers) AverageRating(ctx context.Context, viewer, target string) (*float64, error) {
	db := m.collection.Database()
	if viewer != target {
		if err := db.Collection("ride_groups").FindOne(ctx, bson.M{"members.user_id": bson.M{"$all": []string{viewer, target}}}).Err(); err != nil {
			return nil, err
		}
	}
	cursor, err := db.Collection("ratings").Aggregate(ctx, bson.A{bson.M{"$match": bson.M{"rated_user_id": target}}, bson.M{"$group": bson.M{"_id": nil, "average": bson.M{"$avg": "$rating"}}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if !cursor.Next(ctx) {
		return nil, cursor.Err()
	}
	var result struct {
		Average float64 `bson:"average"`
	}
	if err = cursor.Decode(&result); err != nil {
		return nil, err
	}
	return &result.Average, nil
}
func (a *AuthDependencies) ratings(action string) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		if !user.Onboarded {
			writeJSON(w, 403, map[string]string{"detail": "Complete onboarding first"})
			return
		}
		repository, ok := a.Users.(RatingRepository)
		if !ok {
			writeJSON(w, 503, map[string]string{"detail": "Rating storage unavailable"})
			return
		}
		switch action {
		case "submit":
			var input RatingInput
			if decodeBody(r, &input) != nil || input.Target == "" || input.Target == user.ID || len(input.Target) > 100 || input.GroupID == "" || len(input.GroupID) > 100 || input.Stars < 1 || input.Stars > 5 || input.Comment != nil && utf8.RuneCountInString(*input.Comment) > 200 {
				writeJSON(w, 422, map[string]string{"detail": "Provide a ride partner and a rating from 1 to 5"})
				return
			}
			if input.Comment != nil {
				comment := strings.TrimSpace(*input.Comment)
				input.Comment = &comment
			}
			record, err := repository.SubmitRating(r.Context(), user.ID, input)
			if err != nil {
				ratingFailure(w, err)
				return
			}
			writeJSON(w, 200, record)
		case "pending":
			id := r.PathValue("group_id")
			groups, err := repository.PendingRatings(r.Context(), user.ID, id)
			if err != nil {
				ratingFailure(w, err)
				return
			}
			if id == "" {
				ids := []string{}
				for _, group := range groups {
					ids = append(ids, group.ID)
				}
				writeJSON(w, 200, ids)
				return
			}
			if len(groups) == 0 {
				writeJSON(w, 404, map[string]string{"detail": "Completed ride not found"})
				return
			}
			writeJSON(w, 200, map[string]any{"pending_users": groups[0].Users, "is_only_person": groups[0].Members <= 1, "message": "Rate your partners after completing your trip"})
		case "average":
			average, err := repository.AverageRating(r.Context(), user.ID, r.PathValue("user_id"))
			if err != nil {
				ratingFailure(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"average_rating": average})
		}
	}
}
func ratingFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Ride or partner not found"})
		return
	}
	if errors.Is(err, ErrRatingConflict) {
		writeJSON(w, 409, map[string]string{"detail": "Rating is not available or was already submitted differently"})
		return
	}
	writeJSON(w, 503, map[string]string{"detail": "Unable to load or save rating"})
}
