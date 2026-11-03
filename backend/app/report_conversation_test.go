package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReportParticipantPrivacy(t *testing.T) {
	report := ReportRecord{Reporter: "reporter", Accused: "accused"}
	for _, test := range []struct {
		id, role    string
		admin, want bool
	}{
		{"reporter", "user", false, true}, {"accused", "user", false, false}, {"stranger", "user", false, false},
		{"moderator", "admin", true, true}, {"reporter", "admin", true, false}, {"accused", "admin", true, false}, {"stranger", "user", true, false},
	} {
		if got := reportParticipant(report, User{ID: test.id, Role: test.role}, test.admin); got != test.want {
			t.Fatalf("participant %+v got %v", test, got)
		}
	}
}

type conversationStore struct {
	fakeUsers
	calls   int
	content string
}

func (s *conversationStore) ReportConversation(_ context.Context, u User, id string, admin bool, clientID, content string) (ReportConversation, error) {
	s.calls++
	s.content = content
	return ReportConversation{Report: ReportRecord{ID: id, Reporter: u.ID}, Messages: []ReportMessage{}}, nil
}
func TestReportRepliesRejectSpoofingAndEmptyText(t *testing.T) {
	store := &conversationStore{}
	auth := AuthDependencies{Users: store}
	for _, body := range []string{`{}`, `{"client_message_id":"valid-id","content":" "}`, `{"client_message_id":"short","content":"hello"}`, `{"client_message_id":"valid-id","content":"hello","is_admin":true}`, `{"client_message_id":"valid-id","content":"hello","sender_id":"other"}`} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/reports/r/messages", strings.NewReader(body))
		auth.reportConversation(false)(w, r, User{ID: "reporter"})
		if w.Code != 422 {
			t.Fatalf("got %d for %s", w.Code, body)
		}
	}
	if store.calls != 0 {
		t.Fatal("invalid reply reached storage")
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/reports/r/messages", strings.NewReader(`{"client_message_id":"valid-id","content":" hello "}`))
	r.SetPathValue("report_id", "r")
	auth.reportConversation(false)(w, r, User{ID: "reporter"})
	if w.Code != 200 || store.content != "hello" {
		t.Fatal(w.Code, store.content)
	}
}
func TestEmailLookupRanking(t *testing.T) {
	contacts := []KnownPartner{{ID: "1", Name: "NotEmail", Email: "maya@example.com"}, {ID: "2", Name: "Maya", Email: "arjun@example.com"}, {ID: "3", Email: "mayank@example.com"}}
	if found := rankPartnerEmails(contacts, "maya@example.com"); len(found) == 0 || found[0].ID != "1" {
		t.Fatal(found)
	}
	if found := rankPartnerEmails(contacts, "myaa"); len(found) == 0 || found[0].ID != "1" {
		t.Fatal(found)
	}
	if found := rankPartnerEmails(contacts, "NotEmail"); len(found) != 0 {
		t.Fatal("matched display name", found)
	}
	if found := rankPartnerEmails(contacts, "zzzzzzzz"); len(found) != 0 {
		t.Fatal(found)
	}
}
