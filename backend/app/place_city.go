package server

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Coordinates stay in the request body and are not persisted as location history.
func (p *PlaceSearch) city(w http.ResponseWriter, r *http.Request, user User) {
	w.Header().Set("Cache-Control", "no-store")
	if !user.Onboarded {
		writeJSON(w, 403, map[string]string{"detail": "Complete onboarding first"})
		return
	}
	var point struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if decodeBody(r, &point) != nil || math.IsNaN(point.Lat) || math.IsNaN(point.Lng) || math.IsInf(point.Lat, 0) || math.IsInf(point.Lng, 0) || math.Abs(point.Lat) > 90 || math.Abs(point.Lng) > 180 {
		writeJSON(w, 422, map[string]string{"detail": "Invalid location"})
		return
	}
	if p == nil || !strings.EqualFold(p.base.Hostname(), "api.geoapify.com") || p.token == "" {
		writeJSON(w, 503, map[string]string{"detail": "City lookup unavailable"})
		return
	}
	endpoint := *p.base
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/reverse"
	endpoint.RawQuery = url.Values{"lat": {strconv.FormatFloat(point.Lat, 'f', 4, 64)}, "lon": {strconv.FormatFloat(point.Lng, 'f', 4, 64)}, "type": {"city"}, "format": {"json"}, "apiKey": {p.token}}.Encode()
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint.String(), nil)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "City lookup unavailable"})
		return
	}
	response, err := p.client.Do(request)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "City lookup unavailable"})
		return
	}
	defer response.Body.Close()
	var result struct {
		Results []struct {
			City string `json:"city"`
		} `json:"results"`
	}
	if response.StatusCode != 200 || json.NewDecoder(io.LimitReader(response.Body, 256<<10)).Decode(&result) != nil || len(result.Results) == 0 || strings.TrimSpace(result.Results[0].City) == "" {
		writeJSON(w, 503, map[string]string{"detail": "City lookup unavailable"})
		return
	}
	writeJSON(w, 200, map[string]string{"city": result.Results[0].City})
}
