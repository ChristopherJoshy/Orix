// Package liftschedule computes daily driver availability in the driver's IANA
// timezone. It does not create requests or reserve seats.
package liftschedule

import (
	"errors"
	"time"
)

var ErrInvalidSchedule = errors.New("invalid recurring lift schedule")

// Schedule uses local wall-clock minutes after midnight and a nonempty set of
// weekdays. End is exclusive; overnight windows are deliberately disallowed.
type Schedule struct {
	TimeZone    string         `json:"time_zone" bson:"time_zone"`
	StartMinute int            `json:"start_minute" bson:"start_minute"`
	EndMinute   int            `json:"end_minute" bson:"end_minute"`
	Weekdays    []time.Weekday `json:"weekdays" bson:"weekdays"`
}

// Next returns the first strictly future start and end in UTC, searching at
// most fifteen local calendar days. Nonexistent DST times are skipped rather than
// silently shifted; repeated times use Go's timezone interpretation.
func (s Schedule) Next(after time.Time) (time.Time, time.Time, error) {
	if s.StartMinute < 0 || s.EndMinute > 1440 || s.EndMinute <= s.StartMinute || len(s.Weekdays) == 0 || len(s.Weekdays) > 7 {
		return time.Time{}, time.Time{}, ErrInvalidSchedule
	}
	loc, err := time.LoadLocation(s.TimeZone)
	if err != nil || s.TimeZone == "" || s.TimeZone == "Local" {
		return time.Time{}, time.Time{}, ErrInvalidSchedule
	}
	days := make(map[time.Weekday]bool, len(s.Weekdays))
	for _, day := range s.Weekdays {
		if day < time.Sunday || day > time.Saturday || days[day] {
			return time.Time{}, time.Time{}, ErrInvalidSchedule
		}
		days[day] = true
	}
	local := after.In(loc)
	base := time.Date(local.Year(), local.Month(), local.Day(), 12, 0, 0, 0, loc)
	for i := 0; i < 15; i++ {
		day := base.AddDate(0, 0, i)
		if !days[day.Weekday()] {
			continue
		}
		start := time.Date(day.Year(), day.Month(), day.Day(), s.StartMinute/60, s.StartMinute%60, 0, 0, loc)
		end := time.Date(day.Year(), day.Month(), day.Day(), s.EndMinute/60, s.EndMinute%60, 0, 0, loc)
		if start.Year() != day.Year() || start.Month() != day.Month() || start.Day() != day.Day() || start.Hour()*60+start.Minute() != s.StartMinute {
			continue
		}
		if s.EndMinute < 1440 && (end.Year() != day.Year() || end.Month() != day.Month() || end.Day() != day.Day() || end.Hour()*60+end.Minute() != s.EndMinute) {
			continue
		}
		// A fixed-offset ride request cannot faithfully represent a local window
		// that crosses a clock change. Skip it instead of changing its duration.
		_, startOffset := start.Zone()
		_, endOffset := end.Zone()
		if startOffset != endOffset {
			continue
		}
		if start.After(after) && end.After(start) {
			return start.UTC(), end.UTC(), nil
		}
	}
	return time.Time{}, time.Time{}, ErrInvalidSchedule
}
