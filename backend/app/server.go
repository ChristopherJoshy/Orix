// Package server owns HTTP transport for the Go backend migration.
package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

const Version = "2.1.2"

type Config struct {
	Places            *PlaceSearch
	DeletionWeb       DeletionWebConfig
	TrustedProxies    []netip.Prefix
	AdminSecret       string
	Context           context.Context
	Auth              *AuthDependencies
	AllowedOrigins    []string
	RequestsPerMinute int
	MaxClients        int
}

type clientWindow struct {
	start time.Time
	count int
}
type limiter struct {
	mu                sync.Mutex
	clients           map[string]clientWindow
	limit, maxClients int
}

// allow atomically bounds requests per resolved client address and memory use.
func (l *limiter) allow(peer string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	window, exists := l.clients[peer]
	if !exists {
		if len(l.clients) >= l.maxClients {
			for key, w := range l.clients {
				if now.Sub(w.start) >= time.Minute {
					delete(l.clients, key)
				}
			}
			if len(l.clients) >= l.maxClients {
				return false
			}
		}
		window.start = now
	} else if now.Sub(window.start) >= time.Minute {
		window = clientWindow{start: now}
	}
	if window.count >= l.limit {
		return false
	}
	window.count++
	l.clients[peer] = window
	return true
}

func New(config Config, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if config.RequestsPerMinute <= 0 {
		config.RequestsPerMinute = 60
	}
	if config.MaxClients <= 0 {
		config.MaxClients = 10000
	}
	limits := &limiter{clients: make(map[string]clientWindow), limit: config.RequestsPerMinute, maxClients: config.MaxClients}
	origins := make(map[string]bool, len(config.AllowedOrigins))
	for _, origin := range config.AllowedOrigins {
		if origin != "" && origin != "*" {
			origins[origin] = true
		}
	}
	parent := config.Context
	if parent == nil {
		parent = context.Background()
	}
	hub := &socketHub{clients: make(map[*socketClient]bool)}
	if config.Auth != nil {
		if source, ok := config.Auth.Users.(RealtimeSource); ok {
			go hub.run(parent, source)
		}
	}
	patterns := []string{}
	for _, origin := range config.AllowedOrigins {
		if parsed, err := url.Parse(origin); err == nil && parsed.Host != "" {
			patterns = append(patterns, parsed.Host)
		}
	}
	if config.Auth != nil {
		if worker, ok := config.Auth.Users.(NotificationWorker); ok {
			go worker.RunNotifications(parent)
		}
	}
	if config.Auth != nil {
		if worker, ok := config.Auth.Users.(DeletionRepository); ok {
			go worker.RunDeletions(parent)
		}
	}
	if config.Auth != nil {
		if worker, ok := config.Auth.Users.(MatchingWorker); ok {
			go worker.RunMatchmaking(parent)
		}
	}
	if config.Auth != nil {
		if worker, ok := config.Auth.Users.(LifecycleWorker); ok {
			go worker.RunLifecycle(parent)
		}
	}
	if config.Auth != nil {
		if worker, ok := config.Auth.Users.(interface{ RunDownloadCleanup(context.Context) }); ok {
			go worker.RunDownloadCleanup(parent)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/donations/leaderboard", config.Auth.authenticated(config.Auth.donationLeaders))
	mux.HandleFunc("GET /api/v1/admin/donation-leaderboard", config.Auth.admin(config.AdminSecret, config.Auth.adminDonationLeaderboard))
	mux.HandleFunc("PUT /api/v1/admin/donation-leaderboard", config.Auth.admin(config.AdminSecret, config.Auth.adminDonationLeaderboard))
	mux.HandleFunc("GET /api/v1/public/settings", config.Auth.publicSettings)
	mux.HandleFunc("GET /downloads/android.apk", config.Auth.downloadAPK)
	mux.HandleFunc("PUT /api/v1/admin/public-settings", config.Auth.admin(config.AdminSecret, config.Auth.setPublicSettings))
	mux.HandleFunc("POST /api/v1/admin/android-upload-ticket", config.Auth.admin(config.AdminSecret, config.Auth.apkTicket))
	mux.HandleFunc("PUT /api/v1/admin/android-apk", config.Auth.authenticated(config.Auth.uploadAPK))
	mux.HandleFunc("POST /api/v1/groups/{group_id}/reject", config.Auth.authenticated(config.Auth.rejectGroup))
	mux.HandleFunc("GET /api/v1/places/search", config.Auth.authenticated(config.Places.handler))
	mux.HandleFunc("POST /api/v1/places/city", config.Auth.authenticated(config.Places.city))
	registerDeletionWeb(mux, config.DeletionWeb)
	mux.HandleFunc("GET /readyz", config.Auth.readiness)
	mux.HandleFunc("GET /api/v1/admin/users/lookup", config.Auth.admin(config.AdminSecret, config.Auth.adminRead("lookup")))
	mux.HandleFunc("GET /api/v1/admin/users/{user_id}/avatar", config.Auth.admin(config.AdminSecret, config.Auth.adminAvatar))
	mux.HandleFunc("GET /api/v1/admin/people", config.Auth.admin(config.AdminSecret, config.Auth.adminDirectory("people")))
	mux.HandleFunc("GET /api/v1/admin/groups", config.Auth.admin(config.AdminSecret, config.Auth.adminDirectory("groups")))
	mux.HandleFunc("GET /api/v1/admin/groups/{group_id}/chat", config.Auth.admin(config.AdminSecret, config.Auth.adminDirectory("chat")))
	mux.HandleFunc("POST /api/v1/admin/groups/{group_id}/chat", config.Auth.admin(config.AdminSecret, config.Auth.adminDirectory("send")))
	mux.HandleFunc("GET /api/v1/admin/groups/{group_id}", config.Auth.admin(config.AdminSecret, config.Auth.adminRead("group")))
	mux.HandleFunc("GET /api/v1/admin/users/{user_id}/history", config.Auth.admin(config.AdminSecret, config.Auth.adminRead("logs")))
	mux.HandleFunc("GET /api/v1/admin/logs", config.Auth.admin(config.AdminSecret, config.Auth.adminRead("logs")))
	mux.HandleFunc("GET /api/v1/admin/stats", config.Auth.admin(config.AdminSecret, config.Auth.adminRead("stats")))
	mux.HandleFunc("POST /api/v1/admin/groups/{group_id}/cancel", config.Auth.admin(config.AdminSecret, config.Auth.cancelGroupAsAdmin))
	mux.HandleFunc("GET /api/v1/users/me/terms-status", config.Auth.authenticated(config.Auth.terms))
	mux.HandleFunc("POST /api/v1/users/me/accept-terms", config.Auth.authenticated(config.Auth.terms))
	mux.HandleFunc("POST /api/v1/admin/users/{user_id}/restore", config.Auth.admin(config.AdminSecret, config.Auth.restoreAccount))
	mux.HandleFunc("PUT /api/v1/rides/{request_id}", config.Auth.authenticated(config.Auth.editRide))
	mux.HandleFunc("PUT /api/v1/admin/users/{user_id}/role", config.Auth.admin(config.AdminSecret, config.Auth.changeRole))
	mux.HandleFunc("GET /api/v1/users/search", config.Auth.authenticated(config.Auth.searchPartners))
	mux.HandleFunc("GET /api/v1/app/status", config.Auth.serviceStatus)
	mux.HandleFunc("GET /api/v1/admin/maintenance", config.Auth.admin(config.AdminSecret, func(w http.ResponseWriter, r *http.Request, user User) { config.Auth.serviceStatus(w, r) }))
	mux.HandleFunc("POST /api/v1/admin/maintenance", config.Auth.admin(config.AdminSecret, config.Auth.changeMaintenance))
	mux.HandleFunc("POST /api/v1/users/me/avatar", config.Auth.authenticated(config.Auth.uploadAvatar))
	mux.HandleFunc("GET /api/v1/users/{user_id}/avatar", config.Auth.authenticated(config.Auth.getAvatar))
	mux.HandleFunc("GET /api/v1/admin/jobs", config.Auth.admin(config.AdminSecret, config.Auth.operations("status")))
	mux.HandleFunc("GET /api/v1/admin/engagement", config.Auth.admin(config.AdminSecret, config.Auth.engagementSettings))
	mux.HandleFunc("PUT /api/v1/admin/engagement", config.Auth.admin(config.AdminSecret, config.Auth.engagementSettings))
	mux.HandleFunc("GET /api/v1/admin/notifications/failed", config.Auth.admin(config.AdminSecret, config.Auth.operations("failed")))
	mux.HandleFunc("POST /api/v1/admin/notifications/{event_id}/retry", config.Auth.admin(config.AdminSecret, config.Auth.operations("retry")))
	mux.HandleFunc("DELETE /api/v1/users/me", config.Auth.deleteAccountEndpoint)
	mux.HandleFunc("GET /api/v1/app/version", config.Auth.appVersion)
	mux.HandleFunc("PUT /api/v1/admin/app/version", config.Auth.admin(config.AdminSecret, config.Auth.setAppVersion))
	mux.HandleFunc("POST /api/v1/ratings", config.Auth.authenticated(config.Auth.ratings("submit")))
	mux.HandleFunc("GET /api/v1/ratings/pending", config.Auth.authenticated(config.Auth.ratings("pending")))
	mux.HandleFunc("GET /api/v1/ratings/pending/{group_id}", config.Auth.authenticated(config.Auth.ratings("pending")))
	mux.HandleFunc("GET /api/v1/ratings/average/{user_id}", config.Auth.authenticated(config.Auth.ratings("average")))
	mux.HandleFunc("POST /api/v1/support/tickets", config.Auth.supportAuthenticated(config.Auth.support("create", false)))
	mux.HandleFunc("GET /api/v1/support/tickets", config.Auth.supportAuthenticated(config.Auth.support("list", false)))
	mux.HandleFunc("GET /api/v1/support/tickets/{ticket_id}", config.Auth.supportAuthenticated(config.Auth.support("list", false)))
	mux.HandleFunc("GET /api/v1/admin/support/tickets", config.Auth.admin(config.AdminSecret, config.Auth.support("list", true)))
	mux.HandleFunc("GET /api/v1/admin/support/tickets/{ticket_id}", config.Auth.admin(config.AdminSecret, config.Auth.support("list", true)))
	for _, action := range []string{"message", "close", "reopen"} {
		mux.HandleFunc("POST /api/v1/support/tickets/{ticket_id}/"+action, config.Auth.supportAuthenticated(config.Auth.support(action, false)))
	}
	for _, action := range []string{"message", "close", "reopen", "resolved", "in_progress"} {
		mux.HandleFunc("POST /api/v1/admin/support/tickets/{ticket_id}/"+action, config.Auth.admin(config.AdminSecret, config.Auth.support(action, true)))
	}
	mux.HandleFunc("GET /api/v1/users/me/notification-preferences", config.Auth.authenticated(config.Auth.preferences))
	mux.HandleFunc("PUT /api/v1/users/me/notification-preferences", config.Auth.authenticated(config.Auth.preferences))
	mux.HandleFunc("GET /api/v1/users/me/blocks", config.Auth.authenticated(config.Auth.blocks("list")))
	mux.HandleFunc("PUT /api/v1/users/{user_id}/block", config.Auth.authenticated(config.Auth.blocks("block")))
	mux.HandleFunc("DELETE /api/v1/users/{user_id}/block", config.Auth.authenticated(config.Auth.blocks("unblock")))
	mux.HandleFunc("POST /api/v1/reports", config.Auth.authenticated(config.Auth.reportHandler("create")))
	mux.HandleFunc("GET /api/v1/reports", config.Auth.authenticated(config.Auth.reportHandler("own")))
	mux.HandleFunc("GET /api/v1/reports/{report_id}", config.Auth.authenticated(config.Auth.reportConversation(false)))
	mux.HandleFunc("POST /api/v1/reports/{report_id}/messages", config.Auth.authenticated(config.Auth.reportConversation(false)))
	mux.HandleFunc("GET /api/v1/admin/reports/{report_id}", config.Auth.admin(config.AdminSecret, config.Auth.reportConversation(true)))
	mux.HandleFunc("POST /api/v1/admin/reports/{report_id}/messages", config.Auth.admin(config.AdminSecret, config.Auth.reportConversation(true)))

	mux.HandleFunc("GET /api/v1/admin/reports", config.Auth.admin(config.AdminSecret, config.Auth.reportHandler("admin")))
	mux.HandleFunc("POST /api/v1/admin/reports/{report_id}/resolve", config.Auth.admin(config.AdminSecret, config.Auth.reportHandler("resolve")))
	mux.HandleFunc("GET /api/v1/notifications", config.Auth.authenticated(config.Auth.notificationHandler("list")))
	mux.HandleFunc("PUT /api/v1/notifications/read-all", config.Auth.authenticated(config.Auth.notificationHandler("read")))
	mux.HandleFunc("PUT /api/v1/notifications/{notification_id}/read", config.Auth.authenticated(config.Auth.notificationHandler("read")))
	mux.HandleFunc("DELETE /api/v1/notifications", config.Auth.authenticated(config.Auth.notificationHandler("delete")))
	mux.HandleFunc("DELETE /api/v1/notifications/{notification_id}", config.Auth.authenticated(config.Auth.notificationHandler("delete")))
	mux.HandleFunc("GET /ws", config.Auth.socketHandler(parent, hub, patterns))
	mux.HandleFunc("POST /api/v1/groups/{group_id}/chat", config.Auth.authenticated(config.Auth.sendChat))
	mux.HandleFunc("GET /api/v1/groups/{group_id}/chat", config.Auth.authenticated(config.Auth.readChat))
	mux.HandleFunc("GET /api/v1/groups", config.Auth.authenticated(config.Auth.getGroups))
	mux.HandleFunc("GET /api/v1/groups/{group_id}", config.Auth.authenticated(config.Auth.getGroups))
	for _, action := range []string{"confirm", "mark-here", "start", "finish", "leave"} {
		mux.HandleFunc("POST /api/v1/groups/{group_id}/"+action, config.Auth.authenticated(config.Auth.groupAction(action)))
	}
	mux.HandleFunc("POST /api/v1/rides/{request_id}/group", config.Auth.authenticated(config.Auth.assignGroup))
	mux.HandleFunc("GET /api/v1/rides/{request_id}", config.Auth.authenticated(config.Auth.getRide))
	mux.HandleFunc("GET /api/v1/lifts/recurring", config.Auth.authenticated(config.Auth.recurringLift))
	mux.HandleFunc("PUT /api/v1/lifts/recurring", config.Auth.authenticated(config.Auth.recurringLift))
	mux.HandleFunc("GET /api/v1/rides/status", config.Auth.authenticated(config.Auth.matchingStatus))
	mux.HandleFunc("GET /api/v1/rides/{request_id}/suggestions", config.Auth.authenticated(config.Auth.suggestions))
	mux.HandleFunc("POST /api/v1/rides", config.Auth.authenticated(config.Auth.createRide))
	mux.HandleFunc("GET /api/v1/rides", config.Auth.authenticated(config.Auth.listRides))
	mux.HandleFunc("DELETE /api/v1/rides/{request_id}", config.Auth.authenticated(config.Auth.cancelRide))
	mux.HandleFunc("POST /api/v1/auth/verify", config.Auth.verify)
	mux.HandleFunc("POST /api/v1/auth/onboard", config.Auth.authenticated(config.Auth.onboard))
	mux.HandleFunc("GET /api/v1/users/me", config.Auth.authenticated(config.Auth.getProfile))
	mux.HandleFunc("PUT /api/v1/users/me", config.Auth.authenticated(config.Auth.updateProfile))
	mux.HandleFunc("POST /api/v1/users/me/web-push-token", config.Auth.authenticated(config.Auth.webPushToken))
	mux.HandleFunc("DELETE /api/v1/users/me/web-push-token", config.Auth.authenticated(config.Auth.webPushToken))
	mux.HandleFunc("POST /api/v1/users/me/push-token/unregister", config.Auth.authenticated(config.Auth.unregisterPushToken))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "healthy", "version": Version})
	})
	mux.HandleFunc("GET /api/v1/auth/config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, struct {
			RegistrationOpen bool     `json:"registration_open"`
			AllowedDomains   []string `json:"allowed_domains"`
			Version          string   `json:"version"`
		}{true, []string{}, Version})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("request panic", "method", r.Method)
				writeJSON(w, 500, map[string]string{"detail": "Something went wrong. Please try again."})
			}
		}()
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Add("Vary", "Origin")
			if !origins[origin] {
				writeJSON(w, 403, map[string]string{"detail": "Origin is not allowed"})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Expose-Headers", "X-Next-Cursor")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Secret, X-Upload-Ticket")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		peer := requestPeer(r, config.TrustedProxies)
		// Liveness probes remain available under request pressure.
		if r.URL.Path != "/health" && r.URL.Path != "/readyz" && !limits.allow(peer, time.Now()) {
			w.Header().Set("Retry-After", "60")
			writeJSON(w, 429, map[string]string{"detail": "Too many requests. Please try again shortly."})
			return
		}
		bodyLimit := int64(1 << 20)
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/users/me/avatar" {
			bodyLimit = 3 << 20
		}
		if r.Method == http.MethodPut && r.URL.Path == "/api/v1/admin/android-apk" {
			bodyLimit = maxAPKSize
		}
		r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
		mux.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func ParseOrigins(value string) []string {
	var origins []string
	for _, v := range strings.Split(value, ",") {
		if v = strings.TrimSpace(v); v != "" {
			origins = append(origins, v)
		}
	}
	return origins
}
