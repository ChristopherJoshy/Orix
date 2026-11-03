package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"orix/backend/internal/matching"
)

type RoadStop struct {
	OffsetSeconds *float64  `json:"offset_seconds,omitempty" bson:"offset_seconds,omitempty"`
	UserID        string    `json:"user_id" bson:"user_id"`
	Kind          string    `json:"kind" bson:"kind"`
	Point         []float64 `json:"coordinates" bson:"coordinates"`
}
type RoadPlan struct {
	Earliest time.Time   `json:"earliest_departure" bson:"earliest_departure"`
	Latest   time.Time   `json:"latest_departure" bson:"latest_departure"`
	Stops    []RoadStop  `json:"stops" bson:"stops"`
	Distance float64     `json:"distance_meters" bson:"distance_meters"`
	Duration float64     `json:"duration_seconds" bson:"duration_seconds"`
	Detour   float64     `json:"detour_meters" bson:"detour_meters"`
	Geometry [][]float64 `json:"geometry" bson:"geometry"`
	Created  time.Time   `json:"created_at" bson:"created_at"`
}
type roadCacheEntry struct {
	data    []byte
	expires time.Time
}
type RoadRouter struct {
	mu         sync.Mutex
	cache      map[string]roadCacheEntry
	cacheBytes int
	base       string
	client     *http.Client
}

var ErrNoRoadPlan = errors.New("no road itinerary fits the detour and capacity limits")

func NewRoadRouter(origin string) (*RoadRouter, error) {
	if origin == "" {
		return nil, nil
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" {
		return nil, errors.New("invalid ROUTING_BASE_URL")
	}
	local := parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
	if parsed.Scheme != "https" && !(local && parsed.Scheme == "http") {
		return nil, errors.New("routing requires HTTPS or a loopback HTTP service")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 8
	transport.ResponseHeaderTimeout = 3 * time.Second
	return &RoadRouter{base: strings.TrimRight(origin, "/"), client: &http.Client{Transport: transport, Timeout: 4 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, cache: make(map[string]roadCacheEntry)}, nil
}
func (r *RoadRouter) get(ctx context.Context, path string, result any) error {
	r.mu.Lock()
	cached, found := r.cache[path]
	r.mu.Unlock()
	if found && cached.expires.After(time.Now()) {
		return json.Unmarshal(cached.data, result)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, r.base+path, nil)
	if err != nil {
		return err
	}
	response, err := r.client.Do(request)
	if err != nil {
		return errors.New("road routing is temporarily unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return errors.New("road routing service rejected the itinerary")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return errors.New("road routing response exceeded bounds")
	}
	if err = json.Unmarshal(data, result); err != nil {
		return err
	}
	if len(data) > 512<<10 {
		return nil
	}
	r.mu.Lock()
	if r.cache == nil {
		r.cache = make(map[string]roadCacheEntry)
	}
	if previous, exists := r.cache[path]; exists {
		r.cacheBytes -= len(previous.data)
		delete(r.cache, path)
	}
	for key, entry := range r.cache {
		if entry.expires.Before(time.Now()) {
			r.cacheBytes -= len(entry.data)
			delete(r.cache, key)
		}
	}
	for len(r.cache) >= 256 || r.cacheBytes+len(data) > 32<<20 {
		for key, entry := range r.cache {
			r.cacheBytes -= len(entry.data)
			delete(r.cache, key)
			break
		}
	}
	r.cache[path] = roadCacheEntry{data, time.Now().Add(2 * time.Minute)}
	r.cacheBytes += len(data)
	r.mu.Unlock()
	return nil
}
func coordinates(stops []RoadStop) string {
	parts := make([]string, len(stops))
	for i, stop := range stops {
		parts[i] = fmt.Sprintf("%.6f,%.6f", stop.Point[0], stop.Point[1])
	}
	return strings.Join(parts, ";")
}
func finitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

// The group bound is four travelers. At most 90 legal stop orders remain after
// enforcing pickup-before-drop, with the driver's endpoints fixed.
func bestRoadOrder(distances, durations [][]*float64, seats int, accept ...func([]int) bool) ([]int, float64, float64) {
	n := len(distances)
	if n < 2 || n > 8 || n%2 != 0 || len(durations) != n {
		return nil, 0, 0
	}
	for i := range n {
		if len(distances[i]) != n || len(durations[i]) != n {
			return nil, 0, 0
		}
		for j := range n {
			if distances[i][j] != nil && (!finitePositive(*distances[i][j]) || *distances[i][j] > 40000000) {
				return nil, 0, 0
			}
			if durations[i][j] != nil && (!finitePositive(*durations[i][j]) || *durations[i][j] > 7*24*3600) {
				return nil, 0, 0
			}
		}
	}
	if distances[0][1] == nil || durations[0][1] == nil {
		return nil, 0, 0
	}
	baselineDistance, baselineDuration := *distances[0][1], *durations[0][1]
	if baselineDistance <= 0 || baselineDuration <= 0 {
		return nil, 0, 0
	}
	bestDuration := math.Inf(1)
	bestDistance := 0.0
	var best []int
	var walk func([]int, uint16, int, float64, float64)
	walk = func(order []int, visited uint16, onboard int, distance, duration float64) {
		if duration >= bestDuration || distance > baselineDistance+3000 || duration > baselineDuration+600 {
			return
		}
		if len(order) == n {
			if order[len(order)-1] != 1 {
				return
			}
			// Every passenger's own in-vehicle detour must also fit the policy.
			for pickup := 2; pickup < n; pickup += 2 {
				if distances[pickup][pickup+1] == nil || durations[pickup][pickup+1] == nil {
					return
				}
				riding := false
				rd, rt := 0.0, 0.0
				for i, stop := range order {
					if stop == pickup {
						riding = true
					}
					if stop == pickup+1 {
						break
					}
					if riding && i+1 < len(order) {
						rd += *distances[stop][order[i+1]]
						rt += *durations[stop][order[i+1]]
					}
				}
				if rd > *distances[pickup][pickup+1]+3000 || rt > *durations[pickup][pickup+1]+600 {
					return
				}
			}
			for _, allowed := range accept {
				if !allowed(order) {
					return
				}
			}
			best = append([]int(nil), order...)
			bestDuration = duration
			bestDistance = distance
			return
		}
		from := order[len(order)-1]
		for next := 1; next < n; next++ {
			if visited&(1<<next) != 0 || next == 1 && len(order) != n-1 {
				continue
			}
			passengers := onboard
			if next >= 2 {
				if next%2 == 0 {
					passengers++
				} else {
					if visited&(1<<(next-1)) == 0 {
						continue
					}
					passengers--
				}
			}
			if passengers > seats || passengers < 0 || distances[from][next] == nil || durations[from][next] == nil {
				continue
			}
			walk(append(order, next), visited|(1<<next), passengers, distance+*distances[from][next], duration+*durations[from][next])
		}
	}
	walk([]int{0}, 1, 0, 0, 0)
	return best, bestDistance, bestDuration
}
func (r *RoadRouter) LiftPlan(ctx context.Context, requests []matching.Request) (*RoadPlan, error) {
	if r == nil {
		return nil, ErrRoadRoutingRequired
	}
	if len(requests) < 1 || len(requests) > 4 {
		return nil, ErrNoRoadPlan
	}
	ordered := append([]matching.Request(nil), requests...)
	driver := -1
	for i, request := range ordered {
		if request.Validate() != nil || (request.Mode != matching.Lift && !(request.Mode == matching.Pool && request.AllowLift)) {
			return nil, ErrNoRoadPlan
		}
		if request.Driver {
			if driver >= 0 {
				return nil, ErrNoRoadPlan
			}
			driver = i
		}
	}
	if driver < 0 {
		return nil, ErrNoRoadPlan
	}
	ordered[0], ordered[driver] = ordered[driver], ordered[0]
	stops := make([]RoadStop, 0, len(ordered)*2)
	for _, request := range ordered {
		stops = append(stops, RoadStop{UserID: request.UserID, Kind: "pickup", Point: []float64{request.Pickup.Lng, request.Pickup.Lat}}, RoadStop{UserID: request.UserID, Kind: "dropoff", Point: []float64{request.Drop.Lng, request.Drop.Lat}})
	}
	var table struct {
		Code      string       `json:"code"`
		Distances [][]*float64 `json:"distances"`
		Durations [][]*float64 `json:"durations"`
		Sources   []struct {
			Distance float64 `json:"distance"`
		} `json:"sources"`
	}
	if err := r.get(ctx, "/table/v1/driving/"+coordinates(stops)+"?annotations=distance,duration", &table); err != nil {
		return nil, err
	}
	if table.Code != "Ok" || len(table.Sources) != len(stops) {
		return nil, ErrNoRoadPlan
	}
	for _, source := range table.Sources {
		if !finitePositive(source.Distance) || source.Distance > 1000 {
			return nil, ErrNoRoadPlan
		}
	}
	routingNow := time.Now().UTC()
	order, _, _ := bestRoadOrder(table.Distances, table.Durations, ordered[0].Seats, func(order []int) bool {
		legs := make([]float64, len(order)-1)
		for i := range legs {
			legs[i] = *table.Durations[order[i]][order[i+1]]
		}
		_, _, err := pickupDepartureBounds(order, legs, ordered, routingNow)
		return err == nil
	})
	if len(order) == 0 {
		return nil, ErrNoRoadPlan
	}
	plan := &RoadPlan{Stops: make([]RoadStop, len(order)), Created: time.Now().UTC()}
	for i, index := range order {
		plan.Stops[i] = stops[index]
	}
	var routed struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
			Legs     []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
			} `json:"legs"`
			Geometry struct {
				Type        string      `json:"type"`
				Coordinates [][]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"routes"`
	}
	if err := r.get(ctx, "/route/v1/driving/"+coordinates(plan.Stops)+"?overview=full&geometries=geojson&steps=false", &routed); err != nil {
		return nil, err
	}
	if routed.Code != "Ok" || len(routed.Routes) == 0 {
		return nil, ErrNoRoadPlan
	}
	route := routed.Routes[0]
	if !finitePositive(route.Distance) || !finitePositive(route.Duration) || route.Geometry.Type != "LineString" || len(route.Geometry.Coordinates) < 2 || len(route.Geometry.Coordinates) > 50000 {
		return nil, ErrNoRoadPlan
	}
	baseline := *table.Distances[0][1]
	if route.Distance > baseline+3000 || route.Duration > *table.Durations[0][1]+600 {
		return nil, ErrNoRoadPlan
	}
	for _, point := range route.Geometry.Coordinates {
		if len(point) != 2 || !finitePositive(math.Abs(point[0])) || !finitePositive(math.Abs(point[1])) || math.Abs(point[0]) > 180 || math.Abs(point[1]) > 90 {
			return nil, ErrNoRoadPlan
		}
	}
	if len(route.Legs) != len(order)-1 {
		return nil, ErrNoRoadPlan
	}
	legDurations := make([]float64, len(route.Legs))
	for i, leg := range route.Legs {
		if !finitePositive(leg.Distance) || !finitePositive(leg.Duration) || leg.Duration > 7*24*3600 || leg.Distance > 40000000 {
			return nil, ErrNoRoadPlan
		}
		legDurations[i] = leg.Duration
	}
	totalDistance, totalDuration := 0.0, 0.0
	for _, leg := range route.Legs {
		totalDistance += leg.Distance
		totalDuration += leg.Duration
	}
	if math.Abs(totalDistance-route.Distance) > math.Max(10, route.Distance*.01) || math.Abs(totalDuration-route.Duration) > math.Max(1, route.Duration*.01) {
		return nil, ErrNoRoadPlan
	}
	earliest, latest, boundsErr := pickupDepartureBounds(order, legDurations, ordered, routingNow)
	if boundsErr != nil {
		return nil, boundsErr
	}
	for pickup := 2; pickup < len(stops); pickup += 2 {
		riding := false
		distance, duration := 0.0, 0.0
		for i, stop := range order {
			if stop == pickup {
				riding = true
			}
			if stop == pickup+1 {
				break
			}
			if riding && i < len(route.Legs) {
				distance += route.Legs[i].Distance
				duration += route.Legs[i].Duration
			}
		}
		if distance > *table.Distances[pickup][pickup+1]+3000 || duration > *table.Durations[pickup][pickup+1]+600 {
			return nil, ErrNoRoadPlan
		}
	}
	elapsed := 0.0
	for i := range plan.Stops {
		offset := elapsed
		plan.Stops[i].OffsetSeconds = &offset
		if i < len(legDurations) {
			elapsed += legDurations[i]
		}
	}
	plan.Earliest = earliest
	plan.Latest = latest
	plan.Distance = route.Distance
	plan.Duration = route.Duration
	plan.Detour = math.Max(0, route.Distance-baseline)
	plan.Geometry = route.Geometry.Coordinates
	return plan, nil
}

// Shift each traveler's pickup window back by the time needed to reach it.
func pickupDepartureBounds(order []int, legs []float64, requests []matching.Request, now time.Time) (time.Time, time.Time, error) {
	if len(order) != 2*len(requests) || len(legs) != len(order)-1 {
		return time.Time{}, time.Time{}, ErrNoRoadPlan
	}
	earliest, latest := now, requests[0].End
	elapsed := time.Duration(0)
	for i, stop := range order {
		if stop < 0 || stop/2 >= len(requests) {
			return time.Time{}, time.Time{}, ErrNoRoadPlan
		}
		if stop%2 == 0 {
			request := requests[stop/2]
			start, end := request.Start.Add(-elapsed), request.End.Add(-elapsed)
			if start.After(earliest) {
				earliest = start
			}
			if end.Before(latest) {
				latest = end
			}
		}
		if i < len(legs) {
			if !finitePositive(legs[i]) || legs[i] > 7*24*3600 {
				return time.Time{}, time.Time{}, ErrNoRoadPlan
			}
			elapsed += time.Duration(legs[i] * float64(time.Second))
		}
	}
	if !latest.After(earliest) {
		return time.Time{}, time.Time{}, ErrNoRoadPlan
	}
	return earliest.UTC(), latest.UTC(), nil
}
