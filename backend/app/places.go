package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// PlaceSearch uses Geoapify autocomplete or an operator-selected compatible service. The public
// OSMF endpoint is deliberately excluded: it is not a generic app-search backend.
type PlaceSearch struct {
	base           *url.URL
	token          string
	attribution    string
	attributionURL string
	client         *http.Client
	mu             sync.Mutex
	cache          map[string]placeCache
	next           time.Time
	busy           bool
}
type Place struct {
	ID        string  `json:"place_id"`
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type PlaceResults struct {
	Places         []Place `json:"places"`
	Attribution    string  `json:"attribution"`
	AttributionURL string  `json:"attribution_url"`
}
type placeCache struct {
	result  PlaceResults
	expires time.Time
}

var errPlacesBusy = errors.New("place search busy")

func NewPlaceSearch(origin, token, attribution, attributionURL string) (*PlaceSearch, error) {
	if origin == "" {
		return nil, nil
	}
	base, err := url.Parse(origin)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("invalid GEOCODING_BASE_URL")
	}
	host := strings.TrimSuffix(strings.ToLower(base.Hostname()), ".")
	local := host == "localhost" || host == "127.0.0.1" || host == "::1"
	if host == "nominatim.openstreetmap.org" || (base.Scheme != "https" && !(local && base.Scheme == "http")) {
		return nil, errors.New("geocoding requires a private/contracted HTTPS provider or loopback service")
	}
	credit, err := url.Parse(attributionURL)
	if attribution == "" || len(attribution) > 300 || err != nil || credit.Scheme != "https" || credit.Host == "" || credit.User != nil {
		return nil, errors.New("GEOCODING_ATTRIBUTION and HTTPS GEOCODING_ATTRIBUTION_URL are required")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 1
	transport.ResponseHeaderTimeout = 4 * time.Second
	return &PlaceSearch{base: base, token: token, attribution: attribution, attributionURL: attributionURL,
		cache: make(map[string]placeCache), client: &http.Client{Transport: transport, Timeout: 6 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (p *PlaceSearch) search(ctx context.Context, query string) (PlaceResults, error) {
	key := strings.ToLower(strings.Join(strings.Fields(query), " "))
	p.mu.Lock()
	now := time.Now()
	if cached, ok := p.cache[key]; ok && cached.expires.After(now) {
		p.mu.Unlock()
		return cached.result, nil
	}
	// Bound upstream concurrency and rate without queuing searches behind a slow
	// provider. Operators must still provision the aggregate quota across replicas.
	if p.busy || now.Before(p.next) {
		p.mu.Unlock()
		return PlaceResults{}, errPlacesBusy
	}
	p.busy = true
	interval := 1100 * time.Millisecond
	if strings.EqualFold(p.base.Hostname(), "api.geoapify.com") {
		interval = 250 * time.Millisecond
	}
	p.next = now.Add(interval)
	p.mu.Unlock()
	defer func() { p.mu.Lock(); p.busy = false; p.mu.Unlock() }()
	endpoint := *p.base
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/search"
	endpoint.RawQuery = url.Values{"q": {query}, "format": {"jsonv2"}, "limit": {"5"}}.Encode()
	geoapify := strings.EqualFold(p.base.Hostname(), "api.geoapify.com")
	if geoapify {
		if p.token == "" {
			return PlaceResults{}, errors.New("Geoapify API key is not configured")
		}
		endpoint.Path = strings.TrimRight(p.base.Path, "/") + "/autocomplete"
		endpoint.RawQuery = url.Values{"text": {query}, "format": {"json"}, "limit": {"5"}, "apiKey": {p.token}}.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return PlaceResults{}, err
	}
	request.Header.Set("User-Agent", "ORIX/"+Version)
	request.Header.Set("Accept", "application/json")
	if p.token != "" && !geoapify {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return PlaceResults{}, errors.New("place provider unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return PlaceResults{}, errors.New("place provider unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil || len(body) > 256<<10 {
		return PlaceResults{}, errors.New("place response too large")
	}
	result := PlaceResults{Places: []Place{}, Attribution: p.attribution, AttributionURL: p.attributionURL}
	if geoapify {
		var data struct {
			Results []struct {
				ID    string  `json:"place_id"`
				Label string  `json:"formatted"`
				Lat   float64 `json:"lat"`
				Lng   float64 `json:"lon"`
			} `json:"results"`
		}
		if json.Unmarshal(body, &data) != nil {
			return PlaceResults{}, errors.New("invalid place response")
		}
		for _, row := range data.Results {
			if validPlace(row.Label, row.Lat, row.Lng) && row.ID != "" && len(row.ID) <= 512 {
				result.Places = append(result.Places, Place{ID: "geoapify:" + row.ID, Label: strings.TrimSpace(row.Label), Latitude: row.Lat, Longitude: row.Lng})
			}
			if len(result.Places) == 5 {
				break
			}
		}
	} else {
		var rows []struct {
			Label   string      `json:"display_name"`
			Lat     string      `json:"lat"`
			Lng     string      `json:"lon"`
			OSMType string      `json:"osm_type"`
			OSMID   json.Number `json:"osm_id"`
		}
		if json.Unmarshal(body, &rows) != nil {
			return PlaceResults{}, errors.New("invalid place response")
		}
		for _, row := range rows {
			lat, e1 := strconv.ParseFloat(row.Lat, 64)
			lng, e2 := strconv.ParseFloat(row.Lng, 64)
			if e1 != nil || e2 != nil || !validPlace(row.Label, lat, lng) {
				continue
			}
			id := ""
			if row.OSMType != "" && row.OSMID != "" {
				id = "osm:" + row.OSMType + ":" + row.OSMID.String()
			}
			result.Places = append(result.Places, Place{ID: id, Label: strings.TrimSpace(row.Label), Latitude: lat, Longitude: lng})
			if len(result.Places) == 5 {
				break
			}
		}
	}
	p.mu.Lock()
	if len(p.cache) >= 256 {
		for k, v := range p.cache {
			if !v.expires.After(now) {
				delete(p.cache, k)
			}
		}
		if len(p.cache) >= 256 {
			for k := range p.cache {
				delete(p.cache, k)
				break
			}
		}
	}
	// Searches may contain private addresses: keep only bounded process-memory
	// results, never logs or a durable location-history collection.
	p.cache[key] = placeCache{result: result, expires: now.Add(24 * time.Hour)}
	p.mu.Unlock()
	return result, nil
}

func (p *PlaceSearch) handler(w http.ResponseWriter, r *http.Request, user User) {
	w.Header().Set("Cache-Control", "no-store")
	if !user.Onboarded {
		writeJSON(w, 403, map[string]string{"detail": "Complete onboarding first"})
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if utf8.RuneCountInString(query) < 3 || utf8.RuneCountInString(query) > 200 {
		writeJSON(w, 422, map[string]string{"detail": "Search with 3–200 characters"})
		return
	}
	if p == nil {
		writeJSON(w, 503, map[string]string{"detail": "Place search is unavailable. Select a map point or enter coordinates."})
		return
	}
	result, err := p.search(r.Context(), query)
	if errors.Is(err, errPlacesBusy) {
		w.Header().Set("Retry-After", "2")
		writeJSON(w, 429, map[string]string{"detail": "Search is busy. Try again in a moment."})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Could not search places. Select a map point or try again."})
		return
	}
	writeJSON(w, 200, result)
}

func validPlace(label string, lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && !math.IsInf(lat, 0) && !math.IsInf(lng, 0) && math.Abs(lat) <= 90 && math.Abs(lng) <= 180 && strings.TrimSpace(label) != "" && utf8.RuneCountInString(label) <= 1000
}
