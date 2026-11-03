package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestPublicAuthConfig(t *testing.T) {
	h := New(Config{}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/auth/config", nil))
	var data struct {
		RegistrationOpen bool     `json:"registration_open"`
		Domains          []string `json:"allowed_domains"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &data) != nil || !data.RegistrationOpen || data.Domains == nil || len(data.Domains) != 0 {
		t.Fatalf("unexpected config: %s", w.Body.String())
	}
}
func TestCORSAndRateLimit(t *testing.T) {
	h := New(Config{AllowedOrigins: []string{"https://orix.example"}, RequestsPerMinute: 1}, nil)
	cases := []struct {
		method, origin, path string
		status               int
	}{
		{"OPTIONS", "https://orix.example", "/api/v1/auth/config", 204},
		{"GET", "https://evil.example", "/api/v1/auth/config", 403},
		{"GET", "https://orix.example", "/api/v1/auth/config", 200},
		{"GET", "", "/api/v1/auth/config", 429},
		{"GET", "", "/health", 200},
	}
	for _, tt := range cases {
		r := httptest.NewRequest(tt.method, tt.path, nil)
		r.Header.Set("Origin", tt.origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tt.status {
			t.Fatalf("%s %s: %d", tt.method, tt.path, w.Code)
		}
	}
}
func TestLimiterConcurrentBoundedAndExpires(t *testing.T) {
	l := &limiter{clients: make(map[string]clientWindow), limit: 1, maxClients: 1}
	now := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for range 32 {
		wg.Go(func() {
			if l.allow("a", now) {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if accepted != 1 || l.allow("b", now) || !l.allow("b", now.Add(time.Minute)) {
		t.Fatal("limiter failed atomicity, bound or expiry")
	}
}
func TestForwardedHeaderCannotBypassLimit(t *testing.T) {
	h := New(Config{RequestsPerMinute: 1}, nil)
	for i, ip := range []string{"1.1.1.1", "2.2.2.2"} {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/config", nil)
		r.Header.Set("X-Forwarded-For", ip)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 200
		if i == 1 {
			want = 429
		}
		if w.Code != want {
			t.Fatal(w.Code)
		}
	}
}
