package liftschedule

import (
	"errors"
	"testing"
	"time"
)

func TestNextWeeklyDeparture(t *testing.T) {
	s := Schedule{TimeZone: "Asia/Kolkata", StartMinute: 9 * 60, EndMinute: 9*60 + 30, Weekdays: []time.Weekday{time.Monday}}
	// 09:00 Monday in India is 03:30 UTC; the next call must not repeat it.
	before := time.Date(2026, time.November, 2, 3, 0, 0, 0, time.UTC)
	start, end, err := s.Next(before)
	if err != nil || !start.Equal(before.Add(30*time.Minute)) || end.Sub(start) != 30*time.Minute {
		t.Fatalf("unexpected first departure: %v %v %v", start, end, err)
	}
	next, _, err := s.Next(start)
	if err != nil || !next.Equal(start.AddDate(0, 0, 7)) {
		t.Fatalf("departure was repeated or advanced incorrectly: %v %v", next, err)
	}
}

func TestInvalidSchedules(t *testing.T) {
	for _, s := range []Schedule{
		{TimeZone: "Asia/Kolkata", StartMinute: 500, EndMinute: 500, Weekdays: []time.Weekday{time.Monday}},
		{TimeZone: "Asia/Kolkata", StartMinute: 500, EndMinute: 600, Weekdays: []time.Weekday{time.Monday, time.Monday}},
		{TimeZone: "Local", StartMinute: 500, EndMinute: 600, Weekdays: []time.Weekday{time.Monday}},
		{TimeZone: "Asia/Kolkata", StartMinute: 500, EndMinute: 600},
	} {
		if _, _, err := s.Next(time.Now()); !errors.Is(err, ErrInvalidSchedule) {
			t.Errorf("expected invalid schedule for %+v, got %v", s, err)
		}
	}
}

func TestNonexistentDSTDepartureIsSkipped(t *testing.T) {
	s := Schedule{TimeZone: "America/New_York", StartMinute: 150, EndMinute: 210, Weekdays: []time.Weekday{time.Sunday}}
	start, _, err := s.Next(time.Date(2026, time.March, 8, 0, 0, 0, 0, time.UTC))
	if err != nil || !start.Equal(time.Date(2026, time.March, 15, 6, 30, 0, 0, time.UTC)) {
		t.Fatalf("expected following Sunday after DST gap, got %v %v", start, err)
	}
}
