package server

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Amounts are cumulative, manually verified totals in integer paise, not payments.
type DonationLeader struct {
	ID          string `json:"id" bson:"id"`
	Name        string `json:"name" bson:"name"`
	AmountPaise int64  `json:"amount_paise" bson:"amount_paise"`
}

type DonationLeaderboard struct {
	Version int64            `json:"version" bson:"version"`
	Entries []DonationLeader `json:"entries" bson:"entries"`
}

type DonationLeaderboardRepository interface {
	GetDonationLeaderboard(context.Context) (DonationLeaderboard, error)
	SaveDonationLeaderboard(context.Context, string, DonationLeaderboard) (DonationLeaderboard, error)
}

var errDonationLeaderboardConflict = errors.New("donation leaderboard changed")
var donationLeaderIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func normalizeDonationLeaderboard(board *DonationLeaderboard) bool {
	if board.Version < 0 || board.Version >= 1<<53-1 || len(board.Entries) > 50 {
		return false
	}
	ids := map[string]bool{}
	for i := range board.Entries {
		entry := &board.Entries[i]
		entry.Name = strings.TrimSpace(entry.Name)
		if !donationLeaderIDPattern.MatchString(entry.ID) || ids[entry.ID] || entry.Name == "" || utf8.RuneCountInString(entry.Name) > 60 || len(entry.Name) > 160 || entry.AmountPaise < 1 || entry.AmountPaise > 1_000_000_000 {
			return false
		}
		for _, c := range entry.Name {
			if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
				return false
			}
		}
		ids[entry.ID] = true
	}
	if board.Entries == nil {
		board.Entries = []DonationLeader{}
	}
	sort.Slice(board.Entries, func(i, j int) bool {
		a, b := board.Entries[i], board.Entries[j]
		if a.AmountPaise != b.AmountPaise {
			return a.AmountPaise > b.AmountPaise
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
	return true
}

func (m mongoUsers) GetDonationLeaderboard(ctx context.Context) (DonationLeaderboard, error) {
	board := DonationLeaderboard{Entries: []DonationLeader{}}
	err := m.collection.Database().Collection("app_config").FindOne(ctx, bson.M{"_id": "donation_leaderboard"}).Decode(&board)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return board, nil
	}
	return board, err
}

func (m mongoUsers) SaveDonationLeaderboard(ctx context.Context, actor string, board DonationLeaderboard) (DonationLeaderboard, error) {
	if !normalizeDonationLeaderboard(&board) {
		return board, errors.New("invalid donation leaderboard")
	}
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return board, err
	}
	defer session.EndSession(ctx)
	next := board
	next.Version++
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		db := m.collection.Database()
		collection := db.Collection("app_config")
		if board.Version == 0 {
			_, err := collection.InsertOne(tx, bson.M{"_id": "donation_leaderboard", "version": next.Version, "entries": next.Entries})
			if mongo.IsDuplicateKeyError(err) {
				return nil, errDonationLeaderboardConflict
			}
			if err != nil {
				return nil, err
			}
		} else {
			result, err := collection.UpdateOne(tx, bson.M{"_id": "donation_leaderboard", "version": board.Version}, bson.M{"$set": next})
			if err != nil {
				return nil, err
			}
			if result.MatchedCount != 1 {
				return nil, errDonationLeaderboardConflict
			}
		}
		// Audit the action, not donor names or payment information.
		_, err := WriteAudit(tx, db, bson.M{"actor_id": actor, "action": "update_donation_leaderboard", "target_id": "donation_leaderboard", "created_at": time.Now().UTC()})
		return nil, err
	})
	return next, err
}

func (a *AuthDependencies) donationLeaders(w http.ResponseWriter, r *http.Request, _ User) {
	repository, ok := a.Users.(DonationLeaderboardRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Leaderboard unavailable"})
		return
	}
	board, err := repository.GetDonationLeaderboard(r.Context())
	if err != nil || !normalizeDonationLeaderboard(&board) {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load leaderboard"})
		return
	}
	// Readers receive only the published names and totals, not management IDs.
	type row struct {
		Name        string `json:"name"`
		AmountPaise int64  `json:"amount_paise"`
	}
	rows := make([]row, 0, len(board.Entries))
	for _, entry := range board.Entries {
		rows = append(rows, row{entry.Name, entry.AmountPaise})
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"entries": rows})
}

func (a *AuthDependencies) adminDonationLeaderboard(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.Users.(DonationLeaderboardRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Leaderboard unavailable"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		board, err := repository.GetDonationLeaderboard(r.Context())
		if err != nil || !normalizeDonationLeaderboard(&board) {
			writeJSON(w, 503, map[string]string{"detail": "Unable to load leaderboard"})
			return
		}
		writeJSON(w, 200, board)
		return
	}
	var input struct {
		DonationLeaderboard
		PublicationConfirmed bool `json:"publication_confirmed"`
	}
	if decodeBody(r, &input) != nil || !input.PublicationConfirmed || !normalizeDonationLeaderboard(&input.DonationLeaderboard) {
		writeJSON(w, 422, map[string]string{"detail": "Check donation totals and publication permission"})
		return
	}
	saved, err := repository.SaveDonationLeaderboard(r.Context(), user.ID, input.DonationLeaderboard)
	if adminAccessDenied(w, err) {
		return
	}
	if errors.Is(err, errDonationLeaderboardConflict) {
		writeJSON(w, 409, map[string]string{"detail": "Leaderboard changed; reload before saving"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to save leaderboard"})
		return
	}
	writeJSON(w, 200, saved)
}
