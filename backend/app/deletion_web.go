package server

import (
	"embed"
	"errors"
	"html/template"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

//go:embed account_deletion.html account_deletion.js account_deletion.css
var deletionAssets embed.FS
var deletionTemplate = template.Must(template.ParseFS(deletionAssets, "account_deletion.html"))

// These Firebase web settings are public client identifiers, never Admin SDK
// credentials. Only this explicit projection is sent to browsers.
type DeletionWebConfig struct {
	APIKey       string `json:"apiKey"`
	AuthDomain   string `json:"authDomain"`
	ProjectID    string `json:"projectId"`
	AppID        string `json:"appId"`
	SupportEmail string `json:"-"`
	PrivacyURL   string `json:"-"`
}

func (c DeletionWebConfig) Enabled() bool {
	return c.APIKey != "" && c.AuthDomain != "" && c.ProjectID != "" && c.AppID != ""
}

var deletionAuthDomain = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9.-]{0,251}[a-zA-Z0-9])?$`)

func (c DeletionWebConfig) Validate() error {
	if c.APIKey != "" || c.AuthDomain != "" || c.AppID != "" {
		if !c.Enabled() {
			return errors.New("account deletion web sign-in requires all Firebase web settings")
		}
		if c.SupportEmail == "" || c.PrivacyURL == "" {
			return errors.New("account deletion web sign-in requires ACCOUNT_SUPPORT_EMAIL and PRIVACY_POLICY_URL")
		}
		if len(c.APIKey) > 256 || len(c.AppID) > 256 || !deletionAuthDomain.MatchString(c.AuthDomain) || strings.Contains(c.AuthDomain, "..") {
			return errors.New("invalid account deletion Firebase web configuration")
		}
	}
	if c.SupportEmail != "" {
		address, err := mail.ParseAddress(c.SupportEmail)
		if err != nil || address.Address != c.SupportEmail || len(c.SupportEmail) > 320 {
			return errors.New("ACCOUNT_SUPPORT_EMAIL must be a plain email address")
		}
	}
	if c.PrivacyURL != "" {
		u, err := url.Parse(c.PrivacyURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || len(c.PrivacyURL) > 2048 {
			return errors.New("PRIVACY_POLICY_URL must be an HTTPS URL")
		}
	}
	return nil
}
func deletionWebHeaders(w http.ResponseWriter, config DeletionWebConfig) {
	// Popup OAuth needs its opener. Do not apply COOP same-origin here.
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	policy := "default-src 'none'; script-src 'self' https://www.gstatic.com https://apis.google.com; style-src 'self'; img-src 'self'; connect-src 'self' https://identitytoolkit.googleapis.com https://securetoken.googleapis.com https://www.googleapis.com https://www.gstatic.com https://apis.google.com; frame-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
	if config.Enabled() {
		origin := "https://" + config.AuthDomain
		policy = strings.Replace(policy, "; frame-src 'none'", " "+origin+"; frame-src "+origin, 1)
	}
	w.Header().Set("Content-Security-Policy", policy)
}
func registerDeletionWeb(mux *http.ServeMux, config DeletionWebConfig) {
	mux.HandleFunc("GET /account-deletion", func(w http.ResponseWriter, r *http.Request) {
		deletionWebHeaders(w, config)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// A missing environment must never render an enabled, nonfunctional button.
		if !config.Enabled() && config.SupportEmail == "" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = deletionTemplate.Execute(w, config)
	})
	mux.HandleFunc("GET /account-deletion/config", func(w http.ResponseWriter, r *http.Request) {
		if !config.Enabled() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "Web sign-in is unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, config)
	})
	for path, asset := range map[string]string{
		"/account-deletion/app.js":    "account_deletion.js",
		"/account-deletion/style.css": "account_deletion.css",
	} {
		mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(asset, ".js") {
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			} else {
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			}
			body, err := deletionAssets.ReadFile(asset)
			if err != nil {
				http.Error(w, "Asset unavailable", http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write(body)
		})
	}
}
