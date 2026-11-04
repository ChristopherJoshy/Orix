package matching

import (
	"testing"
	"time"
)

func ride(id string) Request {
	start := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
	return Request{ID: id, UserID: id, Pickup: Point{10, 76}, Drop: Point{10.1, 76}, Start: start, End: start.Add(time.Hour), Mode: Auto}
}
func TestRankConstraints(t *testing.T) {
	a, b := ride("a"), ride("b")
	tests := []struct {
		name   string
		change func(*Request)
		want   int
	}{
		{"same route", func(r *Request) {}, 1},
		{"along route with different destination", func(r *Request) { r.Pickup.Lat += .02; r.Drop.Lat -= .02 }, 0},
		{"nearby pickup same destination", func(r *Request) { r.Pickup.Lat += .001 }, 1},
		{"pickup outside 300 metres", func(r *Request) { r.Pickup.Lat += .003 }, 0},
		{"different drop nearby", func(r *Request) { r.Drop.Lat += .0001 }, 1},
		{"opposite", func(r *Request) { r.Pickup, r.Drop = r.Drop, r.Pickup }, 0},
		{"parallel distant", func(r *Request) { r.Pickup.Lng += .05; r.Drop.Lng += .05 }, 0},
		{"nonoverlapping time", func(r *Request) { r.Start = r.End; r.End = r.End.Add(time.Hour) }, 0},
		{"safety", func(r *Request) { r.FemaleOnly = true; r.Female = true }, 0},
		{"same user", func(r *Request) { r.UserID = a.UserID }, 0},
		{"different mode", func(r *Request) { r.Mode = Partner }, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := b
			tt.change(&c)
			got, err := Rank(a, []Group{{"g", []Request{c}}}, DefaultPolicy())
			if err != nil || len(got) != tt.want {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}
func TestCommonWindowAndAllMembers(t *testing.T) {
	a, b, c := ride("a"), ride("b"), ride("c")
	b.End = b.Start.Add(20 * time.Minute)
	c.Start = c.Start.Add(30 * time.Minute)
	got, _ := Rank(a, []Group{{"g", []Request{b, c}}}, DefaultPolicy())
	if len(got) != 0 {
		t.Fatal("no common departure")
	}
	c = ride("c")
	c.Pickup.Lng += 1
	got, _ = Rank(a, []Group{{"g", []Request{b, c}}}, DefaultPolicy())
	if len(got) != 0 {
		t.Fatal("must check every member")
	}
}
func TestLiftCapacity(t *testing.T) {
	a, b, c := ride("a"), ride("b"), ride("c")
	a.Mode = Lift
	b.Mode = Lift
	c.Mode = Lift
	b.Driver = true
	b.Seats = 1
	groups := []Group{{"g", []Request{b, c}}}
	got, _ := Rank(a, groups, DefaultPolicy())
	if len(got) != 0 {
		t.Fatal("over capacity")
	}
	groups[0].Members[0].Seats = 2
	got, _ = Rank(a, groups, DefaultPolicy())
	if len(got) != 1 {
		t.Fatal("expected lift")
	}
}
func TestInvalidInput(t *testing.T) {
	a := ride("a")
	a.Pickup.Lat = 91
	if _, err := Rank(a, nil, DefaultPolicy()); err == nil {
		t.Fatal("invalid coordinate accepted")
	}
}

func TestDestinationRadiusAndAllPickupPairs(t *testing.T) {
	a, b, c := ride("a"), ride("b"), ride("c")
	a.DestinationID = "place:1"
	b.DestinationID = "place:2"
	if got, _ := Rank(a, []Group{{"g", []Request{b}}}, DefaultPolicy()); len(got) != 1 {
		t.Fatal("different place IDs at the same coordinates should match")
	}
	b.DestinationID = a.DestinationID
	b.Drop.Lat += .00091
	if got, _ := Rank(a, []Group{{"g", []Request{b}}}, DefaultPolicy()); len(got) != 0 {
		t.Fatal("ID overrode the destination radius")
	}
	b.Drop.Lat = a.Drop.Lat + .00089
	b.DestinationID = "place:2"
	if got, _ := Rank(a, []Group{{"g", []Request{b}}}, DefaultPolicy()); len(got) != 1 {
		t.Fatal("nearby destinations with different IDs should match")
	}
	b = ride("b")
	c = ride("c")
	b.Drop.Lat += .0006
	c.Drop.Lat -= .0006
	if got, _ := Rank(a, []Group{{"g", []Request{b, c}}}, DefaultPolicy()); len(got) != 0 {
		t.Fatal("destinations over 100 m apart matched through a center point")
	}
	b = ride("b")
	b.Pickup.Lat += .002
	c.Pickup.Lat -= .002
	if got, _ := Rank(a, []Group{{"g", []Request{b, c}}}, DefaultPolicy()); len(got) != 0 {
		t.Fatal("members more than 300m apart were accepted through a center point")
	}
}

func TestAutomaticPoolRespectsLiftOptIn(t *testing.T) {
	passenger, driver := ride("passenger"), ride("driver")
	passenger.Mode = Pool
	driver.Mode = Lift
	driver.Driver = true
	driver.Seats = 2
	group := []Group{{ID: "lift", Members: []Request{driver}}}
	if got, _ := Rank(passenger, group, DefaultPolicy()); len(got) != 0 {
		t.Fatal("lift accepted without opt-in")
	}
	passenger.AllowLift = true
	if got, _ := Rank(passenger, group, DefaultPolicy()); len(got) != 1 {
		t.Fatal("compatible opted-in lift was rejected")
	}
	if AssignedMode(passenger, []Request{driver}) != Lift {
		t.Fatal("assigned transport not derived from driver")
	}
	peer := ride("peer")
	peer.Mode = Partner
	if got, _ := Rank(passenger, []Group{{ID: "companions", Members: []Request{peer}}}, DefaultPolicy()); len(got) != 1 {
		t.Fatal("automatic passenger could not join companions")
	}
}

func TestRejectionExclusionsSurviveGroupRecreation(t *testing.T) {
	a, b := ride("a"), ride("b")
	a.IgnoredGroups = []string{"rejected"}
	matches, err := Rank(a, []Group{{ID: "rejected", Members: []Request{b}}}, DefaultPolicy())
	if err != nil || len(matches) != 0 {
		t.Fatalf("rejected group was offered: %v %v", matches, err)
	}
	a.IgnoredUsers = []string{b.UserID}
	for _, pair := range [][2]Request{{a, b}, {b, a}} {
		matches, err = Rank(pair[0], []Group{{ID: "new-group-id", Members: []Request{pair[1]}}}, DefaultPolicy())
		if err != nil || len(matches) != 0 {
			t.Fatalf("recreated group bypassed rejection: %v %v", matches, err)
		}
	}
	c := ride("c")
	matches, err = Rank(a, []Group{{ID: "different", Members: []Request{c}}}, DefaultPolicy())
	if err != nil || len(matches) != 1 {
		t.Fatalf("unrelated match was excluded: %v %v", matches, err)
	}
}
