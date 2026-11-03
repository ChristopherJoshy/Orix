package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// getGeoapify translates the bounded internal OSRM-shaped requests to the
// provider's matrix and route contracts, preserving unreachable matrix cells.
func (r *RoadRouter) getGeoapify(ctx context.Context, path string, result any) error {
	isMatrix := strings.HasPrefix(path, "/table/v1/driving/")
	prefix := "/route/v1/driving/"
	if isMatrix {
		prefix = "/table/v1/driving/"
	}
	if !strings.HasPrefix(path, prefix) {
		return ErrNoRoadPlan
	}
	coordinates, _, _ := strings.Cut(strings.TrimPrefix(path, prefix), "?")
	var points [][]float64
	for _, item := range strings.Split(coordinates, ";") {
		parts := strings.Split(item, ",")
		if len(parts) != 2 {
			return ErrNoRoadPlan
		}
		lng, e1 := strconv.ParseFloat(parts[0], 64)
		lat, e2 := strconv.ParseFloat(parts[1], 64)
		if e1 != nil || e2 != nil || !finitePositive(math.Abs(lng)) || !finitePositive(math.Abs(lat)) || math.Abs(lng) > 180 || math.Abs(lat) > 90 {
			return ErrNoRoadPlan
		}
		points = append(points, []float64{lng, lat})
	}
	if len(points) < 2 || len(points) > 8 {
		return ErrNoRoadPlan
	}
	query := url.Values{"apiKey": {r.geoapifyKey}}
	method, endpoint := http.MethodGet, "/routing"
	var body io.Reader
	if isMatrix {
		method, endpoint = http.MethodPost, "/routematrix"
		locations := make([]map[string][]float64, len(points))
		for i, p := range points {
			locations[i] = map[string][]float64{"location": p}
		}
		payload, err := json.Marshal(map[string]any{"mode": "drive", "sources": locations, "targets": locations})
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	} else {
		waypoints := make([]string, len(points))
		for i, p := range points {
			waypoints[i] = fmt.Sprintf("%.6f,%.6f", p[1], p[0])
		}
		query.Set("waypoints", strings.Join(waypoints, "|"))
		query.Set("mode", "drive")
	}
	request, err := http.NewRequestWithContext(ctx, method, r.base+endpoint+"?"+query.Encode(), body)
	if err != nil {
		return err
	}
	if isMatrix {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := r.client.Do(request)
	if err != nil {
		return errors.New("road routing is temporarily unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("road routing service rejected the itinerary")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return errors.New("road routing response exceeded bounds")
	}
	var normalized any
	if isMatrix {
		var matrix struct {
			Sources []struct {
				Original []float64 `json:"original_location"`
				Location []float64 `json:"location"`
			} `json:"sources"`
			Rows [][]struct {
				Distance *float64 `json:"distance"`
				Time     *float64 `json:"time"`
			} `json:"sources_to_targets"`
		}
		if json.Unmarshal(data, &matrix) != nil || len(matrix.Sources) != len(points) || len(matrix.Rows) != len(points) {
			return ErrNoRoadPlan
		}
		distances := make([][]*float64, len(points))
		durations := make([][]*float64, len(points))
		sources := make([]map[string]float64, len(points))
		for i, row := range matrix.Rows {
			if len(row) != len(points) || len(matrix.Sources[i].Location) != 2 {
				return ErrNoRoadPlan
			}
			distances[i] = make([]*float64, len(points))
			durations[i] = make([]*float64, len(points))
			for j, cell := range row {
				distances[i][j] = cell.Distance
				durations[i][j] = cell.Time
			}
			p := matrix.Sources[i].Location
			if math.IsNaN(p[0]) || math.IsNaN(p[1]) || math.IsInf(p[0], 0) || math.IsInf(p[1], 0) || math.Abs(p[0]) > 180 || math.Abs(p[1]) > 90 {
				return ErrNoRoadPlan
			}
			a, b := points[i], p
			rad := math.Pi / 180
			dlat := (a[1] - b[1]) * rad
			dlng := (a[0] - b[0]) * rad
			h := math.Pow(math.Sin(dlat/2), 2) + math.Cos(a[1]*rad)*math.Cos(b[1]*rad)*math.Pow(math.Sin(dlng/2), 2)
			sources[i] = map[string]float64{"distance": 12742000 * math.Asin(math.Sqrt(math.Min(1, h)))}
		}
		normalized = map[string]any{"code": "Ok", "distances": distances, "durations": durations, "sources": sources}
	} else {
		var route struct {
			Features []struct {
				Geometry struct {
					Type        string        `json:"type"`
					Coordinates [][][]float64 `json:"coordinates"`
				} `json:"geometry"`
				Properties struct {
					Distance float64 `json:"distance"`
					Time     float64 `json:"time"`
					Legs     []struct {
						Distance float64 `json:"distance"`
						Time     float64 `json:"time"`
					} `json:"legs"`
				} `json:"properties"`
			} `json:"features"`
		}
		if json.Unmarshal(data, &route) != nil || len(route.Features) == 0 || route.Features[0].Geometry.Type != "MultiLineString" {
			return ErrNoRoadPlan
		}
		f := route.Features[0]
		var geometry [][]float64
		for _, line := range f.Geometry.Coordinates {
			for _, p := range line {
				if len(geometry) > 0 && len(p) == 2 && len(geometry[len(geometry)-1]) == 2 && p[0] == geometry[len(geometry)-1][0] && p[1] == geometry[len(geometry)-1][1] {
					continue
				}
				geometry = append(geometry, p)
			}
		}
		legs := make([]map[string]float64, len(f.Properties.Legs))
		for i, leg := range f.Properties.Legs {
			legs[i] = map[string]float64{"distance": leg.Distance, "duration": leg.Time}
		}
		normalized = map[string]any{"code": "Ok", "routes": []any{map[string]any{"distance": f.Properties.Distance, "duration": f.Properties.Time, "legs": legs, "geometry": map[string]any{"type": "LineString", "coordinates": geometry}}}}
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, result)
}
