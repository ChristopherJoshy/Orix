// Package matching evaluates candidate groups without storage or network calls.
package matching

import (
	"errors"
	"math"
	"sort"
	"time"
)

type Mode string

const (
	Pool    Mode = "pool"
	Auto    Mode = "auto"
	Partner Mode = "partner"
	Lift    Mode = "lift"
)

type Point struct{ Lat, Lng float64 }
type Request struct {
	IgnoredGroups      []string
	IgnoredUsers       []string
	AllowLift          bool
	ID, UserID         string
	DestinationID      string
	Pickup, Drop       Point
	Start, End         time.Time
	Mode               Mode
	Driver             bool
	Seats              int
	FemaleOnly, Female bool
}
type Group struct {
	ID      string
	Members []Request
}
type Match struct {
	GroupID   string
	Score     float64
	Departure time.Time
}
type Policy struct {
	CorridorKM, MaxDetourKM, MinOverlap float64
	MaxMembers                          int
}

func DefaultPolicy() Policy { return Policy{1, 3, .5, 4} }
func validPoint(p Point) bool {
	return !math.IsNaN(p.Lat) && !math.IsNaN(p.Lng) && math.Abs(p.Lat) <= 90 && math.Abs(p.Lng) <= 180
}
func (r Request) Validate() error {
	if r.ID == "" || r.UserID == "" || !validPoint(r.Pickup) || !validPoint(r.Drop) || r.Start.IsZero() || !r.End.After(r.Start) || distance(r.Pickup, r.Drop) < .1 {
		return errors.New("invalid ride request")
	}
	if r.Mode != Pool && r.Mode != Auto && r.Mode != Partner && r.Mode != Lift {
		return errors.New("invalid travel mode")
	}
	if r.Driver && (r.Mode != Lift || r.Seats < 1 || r.Seats > 8) {
		return errors.New("invalid lift capacity")
	}
	if r.FemaleOnly && !r.Female {
		return errors.New("female-only request requires female profile")
	}
	return nil
}
func rad(d float64) float64 { return d * math.Pi / 180 }
func distance(a, b Point) float64 {
	dlat, dlng := rad(b.Lat-a.Lat), rad(b.Lng-a.Lng)
	h := math.Pow(math.Sin(dlat/2), 2) + math.Cos(rad(a.Lat))*math.Cos(rad(b.Lat))*math.Pow(math.Sin(dlng/2), 2)
	return 6371 * 2 * math.Asin(math.Sqrt(math.Min(1, h)))
}
func bearing(a, b Point) float64 {
	x := rad(b.Lng - a.Lng)
	return math.Atan2(math.Sin(x)*math.Cos(rad(b.Lat)), math.Cos(rad(a.Lat))*math.Sin(rad(b.Lat))-math.Sin(rad(a.Lat))*math.Cos(rad(b.Lat))*math.Cos(x))
}

// position returns cross-track distance and signed distance along a great circle.
func position(a, b, p Point) (float64, float64) {
	d := distance(a, p) / 6371
	angle := bearing(a, p) - bearing(a, b)
	cross := math.Asin(math.Max(-1, math.Min(1, math.Sin(d)*math.Sin(angle)))) * 6371
	along := math.Atan2(math.Sin(d)*math.Cos(angle), math.Cos(d)) * 6371
	return math.Abs(cross), along
}

// PickupRadiusKM is a hard constraint, independent of route scoring.
const PickupRadiusKM = .3

// DestinationRadiusKM applies pairwise to all group members.
const DestinationRadiusKM = .1

func SameDestination(a, b Request) bool {
	// Place IDs and labels may differ for nearby points; coordinates set the limit.
	return distance(a.Drop, b.Drop) <= DestinationRadiusKM
}
func nearbySameDestination(a, b Request) bool {
	return distance(a.Pickup, b.Pickup) <= PickupRadiusKM && SameDestination(a, b)
}

func route(a, b Request, p Policy) float64 {
	length := distance(a.Pickup, a.Drop)
	cross1, start := position(a.Pickup, a.Drop, b.Pickup)
	cross2, end := position(a.Pickup, a.Drop, b.Drop)
	if cross1 > p.CorridorKM || cross2 > p.CorridorKM || end <= start {
		return 0
	}
	shared := math.Max(0, math.Min(length, end)-math.Max(0, start))
	overlap := shared / math.Min(length, distance(b.Pickup, b.Drop))
	detour := distance(a.Pickup, b.Pickup) + distance(b.Pickup, b.Drop) + distance(b.Drop, a.Drop) - length
	// Either journey can be the longer shared corridor.
	reverseDetour := distance(b.Pickup, a.Pickup) + length + distance(a.Drop, b.Drop) - distance(b.Pickup, b.Drop)
	if overlap < p.MinOverlap || math.Min(detour, reverseDetour) > p.MaxDetourKM {
		return 0
	}
	return math.Min(1, overlap) / (1 + math.Max(cross1, cross2))
}

func compatibleMode(a, b Request) bool {
	if a.Mode == b.Mode {
		return true
	}
	if a.Mode == Pool {
		return b.Mode != Lift || a.AllowLift
	}
	if b.Mode == Pool {
		return a.Mode != Lift || b.AllowLift
	}
	return false
}
func RequiresLift(request Request, members []Request) bool {
	if request.Mode == Lift {
		return true
	}
	for _, member := range members {
		if member.Mode == Lift {
			return true
		}
	}
	return false
}

// AssignedMode is an outcome. Generic passenger requests never select transport.
func AssignedMode(request Request, members []Request) Mode {
	if RequiresLift(request, members) {
		return Lift
	}
	if request.Mode == Partner {
		return Partner
	}
	for _, member := range members {
		if member.Mode == Partner {
			return Partner
		}
	}
	return Auto
}

// Rank requires a common departure time and compatibility with every member.
// Locations are geometric suggestions; road routing must confirm a lift itinerary.
func Rank(r Request, groups []Group, p Policy) ([]Match, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if p.CorridorKM <= 0 || p.MaxDetourKM <= 0 || p.MinOverlap <= 0 || p.MinOverlap > 1 || p.MaxMembers < 2 || math.IsNaN(p.CorridorKM) || math.IsNaN(p.MaxDetourKM) || math.IsNaN(p.MinOverlap) {
		return nil, errors.New("invalid matching policy")
	}
	matches := make([]Match, 0)
	for _, g := range groups {
		if containsID(r.IgnoredGroups, g.ID) || g.ID == "" || len(g.Members) == 0 || len(g.Members) >= p.MaxMembers {
			continue
		}
		start, end, score, ok := r.Start, r.End, 1.0, true
		drivers, seats := 0, 0
		if r.Driver {
			drivers = 1
			seats = r.Seats
		}
		femaleOnly, allFemale := r.FemaleOnly, r.Female
		users := map[string]bool{r.UserID: true}
		for memberIndex, m := range g.Members {
			for _, other := range g.Members[:memberIndex] {
				if ignoredPair(m, other) || !nearbySameDestination(m, other) || !compatibleMode(m, other) {
					ok = false
					break
				}
			}
			if !ok {
				break
			}
			if ignoredPair(r, m) || m.Validate() != nil || !nearbySameDestination(r, m) || users[m.UserID] || !compatibleMode(m, r) || ((m.FemaleOnly || r.FemaleOnly) && (!m.Female || !r.Female)) {
				ok = false
				break
			}
			users[m.UserID] = true
			femaleOnly = femaleOnly || m.FemaleOnly
			allFemale = allFemale && m.Female
			if m.Driver {
				drivers++
				seats = m.Seats
			}
			if m.Start.After(start) {
				start = m.Start
			}
			if m.End.Before(end) {
				end = m.End
			}
			s := math.Min(route(r, m, p), route(m, r, p))
			if s == 0 {
				ok = false
				break
			}
			score = math.Min(score, s)
		}
		lift := RequiresLift(r, g.Members)
		if !ok || (!end.After(start) && !lift) || drivers > 1 || (femaleOnly && !allFemale) {
			continue
		}
		if lift && (drivers != 1 || len(g.Members) > seats) {
			continue
		}
		matches = append(matches, Match{g.ID, score, start})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].GroupID < matches[j].GroupID
		}
		return matches[i].Score > matches[j].Score
	})
	return matches, nil
}

func containsID(ids []string, value string) bool {
	for _, id := range ids {
		if id == value {
			return true
		}
	}
	return false
}
func ignoredPair(a, b Request) bool {
	return containsID(a.IgnoredUsers, b.UserID) || containsID(b.IgnoredUsers, a.UserID)
}
