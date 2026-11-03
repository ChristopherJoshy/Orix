package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type profileStore struct {
	fakeUsers
	updates      map[string]any
	onboardCalls int
	updatedID    string
}

func (p *profileStore) UpdateProfile(_ context.Context, id string, changes map[string]any) (User, error) {
	p.updatedID = id
	p.updates = changes
	return User{ID: id, Name: "Updated", Status: "active"}, nil
}
func (p *profileStore) CompleteOnboarding(_ context.Context, id, phone string, gender *string) error {
	p.updatedID = id
	p.onboardCalls++
	return nil
}
func requestProfile(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestProfilesRequireVerifiedIdentity(t *testing.T) {
	store := &profileStore{fakeUsers: fakeUsers{status: "active"}}
	h := New(Config{Auth: &AuthDependencies{fakeVerifier{}, store}}, nil)
	for _, token := range []string{"", "Basic abc", "Bearer x y"} {
		w := requestProfile(h, "GET", "/api/v1/users/me", "", token)
		if w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	if store.calls != 0 {
		t.Fatal("anonymous request reached storage")
	}
	w := requestProfile(h, "GET", "/api/v1/users/me", "", "Bearer valid")
	var data map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &data)
	if w.Code != 200 || data["status"] != "active" || data["user_id"] != "u" {
		t.Fatal(w.Body.String())
	}
	for _, field := range []string{"phone", "gender", "is_rider", "rider_info", "onboarding_completed"} {
		if _, ok := data[field]; !ok {
			t.Fatalf("missing %s", field)
		}
	}
}
func TestProfileUpdateIsAllowlisted(t *testing.T) {
	store := &profileStore{fakeUsers: fakeUsers{status: "active"}}
	h := New(Config{Auth: &AuthDependencies{fakeVerifier{}, store}}, nil)
	for _, body := range []string{`{"role":"admin"}`, `{"user_id":"other"}`, `{"display_name":" "}`, `{"photo_url":"http://unsafe.example/a"}`, `{"photo_url":"javascript:alert(1)"}`} {
		if w := requestProfile(h, "PUT", "/api/v1/users/me", body, "Bearer valid"); w.Code != 422 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if store.updates != nil {
		t.Fatal("invalid input persisted")
	}
	w := requestProfile(h, "PUT", "/api/v1/users/me", `{"display_name":" Traveler ","fcm_token":"new-token"}`, "Bearer valid")
	if w.Code != 200 || store.updatedID != "u" || store.updates["display_name"] != "Traveler" {
		t.Fatal(w.Body.String(), store.updates)
	}
	if strings.Contains(w.Body.String(), "new-token") {
		t.Fatal("push token leaked")
	}
}
func TestOnboardingValidationAndConsent(t *testing.T) {
	store := &profileStore{fakeUsers: fakeUsers{status: "active"}}
	h := New(Config{Auth: &AuthDependencies{fakeVerifier{}, store}}, nil)
	for _, tt := range []struct {
		body string
		want int
	}{
		{`{"phone":"+919876543210","gender":"F","safety_consent":false,"privacy_consent":true}`, 400},
		{`{"phone":"+919876543210","gender":"F","safety_consent":true}`, 400},
		{`{"phone":"+919876543210","gender":"F","safety_consent":true,"privacy_consent":false}`, 400},
		{`{"phone":"junk","safety_consent":true}`, 422},
		{`{"phone":"+919876543210","gender":"invalid","safety_consent":true}`, 422},
		{`{"phone":"+919876543210","gender":"F","safety_consent":true,"privacy_consent":true}`, 200},
	} {
		w := requestProfile(h, "POST", "/api/v1/auth/onboard", tt.body, "Bearer valid")
		if w.Code != tt.want {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if store.onboardCalls != 1 || store.updatedID != "u" {
		t.Fatal("invalid completion persisted")
	}
}
func TestRestrictedProfileCannotBeMutated(t *testing.T) {
	store := &profileStore{fakeUsers: fakeUsers{status: "banned"}}
	h := New(Config{Auth: &AuthDependencies{fakeVerifier{}, store}}, nil)
	w := requestProfile(h, "PUT", "/api/v1/users/me", `{"display_name":"Traveler"}`, "Bearer valid")
	if w.Code != 403 || store.updates != nil {
		t.Fatal(w.Code)
	}
}
