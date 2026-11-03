package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPlaceSearchRejectsUnsafeConfiguration(t *testing.T) {
	for _, origin := range []string{"https://nominatim.openstreetmap.org", "https://NOMINATIM.OPENSTREETMAP.ORG./", "http://provider.example", "https://secret@provider.example", "https://provider.example?token=secret"} {
		if _, err := NewPlaceSearch(origin, "", "Provider", "https://provider.example/credit"); err == nil {
			t.Errorf("accepted unsafe origin %q", origin)
		}
	}
	if _, err := NewPlaceSearch("https://provider.example", "", "", ""); err == nil {
		t.Fatal("accepted provider without attribution")
	}
}

func TestPlaceSearchCachesAndBoundsResults(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/search" || r.URL.Query().Get("q") != "City station" || r.URL.Query().Get("limit") != "5" {
			t.Error("incorrect provider request")
		}
		if r.Header.Get("Authorization") != "Bearer test-only" || r.Header.Get("User-Agent") == "" {
			t.Error("missing provider authentication/identity")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"display_name":"City station","lat":"12.3","lon":"45.6"},{"display_name":"Invalid","lat":"NaN","lon":"3"},{"display_name":"Out of range","lat":"91","lon":"0"}]`))
	}))
	defer upstream.Close()
	provider, err := NewPlaceSearch(upstream.URL, "test-only", "Provider", "https://provider.example/credit")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		result, err := provider.search(context.Background(), "City station")
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Places) != 1 || result.Places[0].Latitude != 12.3 || result.Attribution != "Provider" {
			t.Fatalf("incorrect bounded results: %+v", result)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("cached query called upstream again")
	}
	if _, err := provider.search(context.Background(), "Different place"); !errors.Is(err, errPlacesBusy) {
		t.Fatalf("provider quota not enforced: %v", err)
	}
}

func TestPlaceSearchRejectsRedirect(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://provider.example/redirected", http.StatusFound)
	}))
	defer upstream.Close()
	provider, err := NewPlaceSearch(upstream.URL, "", "Provider", "https://provider.example/credit")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.search(context.Background(), "City station"); err == nil {
		t.Fatal("accepted redirected response")
	}
}

func TestGeoapifyAutocompleteAdapter(t *testing.T) {
	provider, err := NewPlaceSearch("https://api.geoapify.com/v1/geocode", "test-key", "Powered by Geoapify | OpenStreetMap contributors", "https://www.geoapify.com/")
	if err != nil {
		t.Fatal(err)
	}
	provider.client = &http.Client{Transport: placeRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/geocode/autocomplete" || r.URL.Query().Get("text") != "Edappally" || r.URL.Query().Get("apiKey") != "test-key" || r.Header.Get("Authorization") != "" {
			t.Fatal("incorrect autocomplete request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[{"place_id":"abc","formatted":"Edappally, Kochi","lat":10.0261,"lon":76.3125}]}`)), Header: make(http.Header)}, nil
	})}
	result, err := provider.search(context.Background(), "Edappally")
	if err != nil || len(result.Places) != 1 || result.Places[0].ID != "geoapify:abc" {
		t.Fatal(result, err)
	}
}

type placeRoundTrip func(*http.Request) (*http.Response, error)

func (f placeRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
