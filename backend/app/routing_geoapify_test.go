package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeoapifyRoutingTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("apiKey") != "test-key" {
			t.Error("missing provider key")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/routematrix":
			if r.Method != http.MethodPost {
				t.Error("matrix must use POST")
			}
			var request struct {
				Sources []struct {
					Location []float64 `json:"location"`
				} `json:"sources"`
			}
			if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.Sources) != 2 || request.Sources[0].Location[0] != 76.28 {
				t.Error("wrong matrix coordinates")
			}
			_, _ = w.Write([]byte(`{"sources":[{"location":[76.28,9.98]},{"location":[76.31,10.01]}],"sources_to_targets":[[{"distance":0,"time":0},{"distance":5000,"time":600}],[{"distance":5100,"time":610},{"distance":0,"time":0}]]}`))
		case "/routing":
			if r.URL.Query().Get("waypoints") != "9.980000,76.280000|10.010000,76.310000" {
				t.Error("wrong route coordinate order")
			}
			_, _ = w.Write([]byte(`{"features":[{"geometry":{"type":"MultiLineString","coordinates":[[[76.28,9.98],[76.29,9.99]],[[76.29,9.99],[76.31,10.01]]]},"properties":{"distance":5000,"time":600,"legs":[{"distance":5000,"time":600}]}}]}`))
		default:
			t.Error("unexpected provider path")
		}
	}))
	defer server.Close()
	r := &RoadRouter{base: server.URL, geoapifyKey: "test-key", client: server.Client()}
	var matrix struct {
		Code      string       `json:"code"`
		Distances [][]*float64 `json:"distances"`
		Sources   []struct {
			Distance float64 `json:"distance"`
		} `json:"sources"`
	}
	if err := r.getGeoapify(context.Background(), "/table/v1/driving/76.280000,9.980000;76.310000,10.010000?annotations=distance,duration", &matrix); err != nil || matrix.Code != "Ok" || len(matrix.Distances) != 2 || *matrix.Distances[0][1] != 5000 || matrix.Sources[0].Distance != 0 {
		t.Fatalf("invalid matrix translation: %+v %v", matrix, err)
	}
	var route struct {
		Code   string `json:"code"`
		Routes []struct {
			Geometry struct {
				Coordinates [][]float64 `json:"coordinates"`
			} `json:"geometry"`
			Legs []struct {
				Duration float64 `json:"duration"`
			} `json:"legs"`
		} `json:"routes"`
	}
	if err := r.getGeoapify(context.Background(), "/route/v1/driving/76.280000,9.980000;76.310000,10.010000?overview=full", &route); err != nil || route.Code != "Ok" || len(route.Routes) != 1 || len(route.Routes[0].Geometry.Coordinates) != 3 || route.Routes[0].Legs[0].Duration != 600 {
		t.Fatalf("invalid route translation: %+v %v", route, err)
	}
}
