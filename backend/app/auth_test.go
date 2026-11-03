package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeVerifier struct{ err error }

func (f fakeVerifier) Verify(context.Context, string) (Identity, error) {
	return Identity{UID: "uid", Email: "traveler@gmail.com"}, f.err
}

type fakeUsers struct {
	calls  int
	status string
	err    error
}

func (f *fakeUsers) GetOrCreate(_ context.Context, i Identity) (User, bool, error) {
	f.calls++
	return User{ID: "u", Email: i.Email, Name: "Traveler", Status: f.status}, true, f.err
}
func TestVerifyContract(t *testing.T) {
	tests := []struct {
		name, body, status          string
		verifierError, storageError error
		want                        int
	}{
		{name: "public email", body: `{"id_token":"valid"}`, status: "active", want: 200},
		{name: "missing token", body: `{}`, want: 422},
		{name: "trailing input", body: `{"id_token":"valid"}{}`, want: 422},
		{name: "unknown field", body: `{"id_token":"valid","role":"admin"}`, want: 422},
		{name: "invalid token", body: `{"id_token":"invalid"}`, verifierError: errors.New("invalid"), want: 401},
		{name: "banned", body: `{"id_token":"valid"}`, status: "banned", want: 403},
		{name: "suspended", body: `{"id_token":"valid"}`, status: "suspended", want: 403},
		{name: "database unavailable", body: `{"id_token":"valid"}`, storageError: errors.New("private database URI"), want: 503},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := &fakeUsers{status: tt.status, err: tt.storageError}
			h := New(Config{Auth: &AuthDependencies{fakeVerifier{tt.verifierError}, users}}, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/auth/verify", strings.NewReader(tt.body)))
			if w.Code != tt.want {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
			if tt.want == 401 || tt.want == 422 {
				if users.calls != 0 {
					t.Fatal("unverified identity reached storage")
				}
			}
			if strings.Contains(w.Body.String(), "private database") {
				t.Fatal("internal error leaked")
			}
			if tt.want == 200 && (!strings.Contains(w.Body.String(), `"is_new_user":true`) || strings.Contains(w.Body.String(), `"status"`)) {
				t.Fatal(w.Body.String())
			}
		})
	}
}
func TestAuthUnconfiguredFailsClosed(t *testing.T) {
	w := httptest.NewRecorder()
	New(Config{}, nil).ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/auth/verify", strings.NewReader(`{"id_token":"x"}`)))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
}
