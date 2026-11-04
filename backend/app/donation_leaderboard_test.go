package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDonationLeaderboardValidation(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*DonationLeaderboard)
	}{
		{"negative version", func(b *DonationLeaderboard) { b.Version = -1 }},
		{"unsafe version", func(b *DonationLeaderboard) { b.Version = 1<<53 - 1 }},
		{"zero amount", func(b *DonationLeaderboard) { b.Entries[0].AmountPaise = 0 }},
		{"negative amount", func(b *DonationLeaderboard) { b.Entries[0].AmountPaise = -1 }},
		{"excessive amount", func(b *DonationLeaderboard) { b.Entries[0].AmountPaise = 1_000_000_001 }},
		{"blank name", func(b *DonationLeaderboard) { b.Entries[0].Name = "  " }},
		{"long name", func(b *DonationLeaderboard) { b.Entries[0].Name = strings.Repeat("a", 61) }},
		{"control character", func(b *DonationLeaderboard) { b.Entries[0].Name = "A\nB" }},
		{"bidi override", func(b *DonationLeaderboard) { b.Entries[0].Name = "A\u202eB" }},
		{"bad ID", func(b *DonationLeaderboard) { b.Entries[0].ID = "a.b" }},
		{"duplicate ID", func(b *DonationLeaderboard) { b.Entries = append(b.Entries, b.Entries[0]) }},
		{"too many entries", func(b *DonationLeaderboard) { b.Entries = make([]DonationLeader, 51) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			board := DonationLeaderboard{Entries: []DonationLeader{{"one", "Supporter", 10000}}}
			test.edit(&board)
			if normalizeDonationLeaderboard(&board) {
				t.Fatal("invalid donation entry accepted")
			}
		})
	}
	board := DonationLeaderboard{Entries: []DonationLeader{{"b", " Second ", 10050}, {"a", "First", 20000}, {"c", "Another", 10050}}}
	if !normalizeDonationLeaderboard(&board) || board.Entries[0].Name != "First" || board.Entries[1].Name != "Another" || board.Entries[2].Name != "Second" {
		t.Fatal("ranking, tie ordering or name normalization failed")
	}
	board = DonationLeaderboard{}
	if !normalizeDonationLeaderboard(&board) || board.Entries == nil {
		t.Fatal("empty board must serialize as a real empty list")
	}
}

// Isolated transport doubles; these do not exercise Firebase or Mongo persistence.
type donationLeaderboardTestUsers struct {
	fakeUsers
	role, status string
	board        DonationLeaderboard
	readErr      error
	saveErr      error
	writes       int
}

func (f *donationLeaderboardTestUsers) GetOrCreate(context.Context, Identity) (User, bool, error) {
	return User{ID: "admin-unit-test", Role: f.role, Status: f.status, Onboarded: true}, false, nil
}
func (f *donationLeaderboardTestUsers) GetDonationLeaderboard(context.Context) (DonationLeaderboard, error) {
	return f.board, f.readErr
}
func (f *donationLeaderboardTestUsers) SaveDonationLeaderboard(_ context.Context, actor string, board DonationLeaderboard) (DonationLeaderboard, error) {
	if actor != "admin-unit-test" {
		return board, errors.New("unexpected actor")
	}
	f.writes++
	board.Version++
	return board, f.saveErr
}

func TestDonationLeaderboardHTTP(t *testing.T) {
	const valid = `{"version":0,"entries":[{"id":"one","name":"Supporter","amount_paise":12550}],"publication_confirmed":true}`
	for _, test := range []struct {
		name, method, path, role, status, token, secret, body string
		readErr, saveErr                                      error
		want, writes                                          int
	}{
		{name: "reader projection", method: "GET", path: "/api/v1/donations/leaderboard", token: "unit", want: 200},
		{name: "signed out", method: "GET", path: "/api/v1/donations/leaderboard", want: 401},
		{name: "non admin read", method: "GET", path: "/api/v1/admin/donation-leaderboard", token: "unit", secret: "test-secret", want: 403},
		{name: "admin read", method: "GET", path: "/api/v1/admin/donation-leaderboard", role: "admin", token: "unit", secret: "test-secret", want: 200},
		{name: "missing secret", method: "PUT", path: "/api/v1/admin/donation-leaderboard", role: "admin", token: "unit", body: valid, want: 403},
		{name: "non admin write", method: "PUT", path: "/api/v1/admin/donation-leaderboard", token: "unit", secret: "test-secret", body: valid, want: 403},
		{name: "banned admin", method: "PUT", path: "/api/v1/admin/donation-leaderboard", role: "admin", status: "banned", token: "unit", secret: "test-secret", body: valid, want: 403},
		{name: "publish", method: "PUT", path: "/api/v1/admin/donation-leaderboard", role: "admin", token: "unit", secret: "test-secret", body: valid, want: 200, writes: 1},
		{name: "consent required", method: "PUT", path: "/api/v1/admin/donation-leaderboard", role: "admin", token: "unit", secret: "test-secret", body: strings.Replace(valid, `true`, `false`, 1), want: 422},
		{name: "fractional paise", method: "PUT", path: "/api/v1/admin/donation-leaderboard", role: "admin", token: "unit", secret: "test-secret", body: strings.Replace(valid, `12550`, `12550.5`, 1), want: 422},
		{name: "stale write", method: "PUT", path: "/api/v1/admin/donation-leaderboard", role: "admin", token: "unit", secret: "test-secret", body: valid, saveErr: errDonationLeaderboardConflict, want: 409, writes: 1},
		{name: "admin demoted during save", method: "PUT", path: "/api/v1/admin/donation-leaderboard", role: "admin", token: "unit", secret: "test-secret", body: valid, saveErr: ErrAdminAccess, want: 403, writes: 1},
		{name: "storage failure", method: "GET", path: "/api/v1/donations/leaderboard", token: "unit", readErr: errors.New("private database address"), want: 503},
	} {
		t.Run(test.name, func(t *testing.T) {
			users := &donationLeaderboardTestUsers{role: test.role, status: test.status, board: DonationLeaderboard{Version: 1, Entries: []DonationLeader{{"private-management-id", "Supporter", 12550}}}, readErr: test.readErr, saveErr: test.saveErr}
			h := New(Config{Auth: &AuthDependencies{fakeVerifier{}, users}, AdminSecret: "test-secret"}, nil)
			r := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.token != "" {
				r.Header.Set("Authorization", "Bearer "+test.token)
			}
			r.Header.Set("X-Admin-Secret", test.secret)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != test.want || users.writes != test.writes {
				t.Fatalf("status %d writes %d: %s", w.Code, users.writes, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "private database") || (test.name == "reader projection" && (strings.Contains(w.Body.String(), "private-management-id") || strings.Contains(w.Body.String(), "version"))) {
				t.Fatal("private detail leaked")
			}
		})
	}
}
