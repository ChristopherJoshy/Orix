package server

import (
	"testing"
	"time"

	"orix/backend/internal/liftschedule"
)

func TestRecurringEditReplayComparison(t *testing.T) {
	allow := true
	base := RecurringLift{Active: true, Input: RideInput{PickupPlaceID: "a", DropPlaceID: "b", AllowRiders: &allow}, Schedule: liftschedule.Schedule{TimeZone: "Asia/Kolkata", StartMinute: 540, EndMinute: 570, Weekdays: []time.Weekday{time.Monday, time.Tuesday}}}
	replay := base
	replay.Input.Date = "2026-11-03"
	replay.Schedule.Weekdays = []time.Weekday{time.Tuesday, time.Monday}
	if !sameRecurringEdit(base, replay) {
		t.Fatal("equivalent retry should not depend on generated date or weekday ordering")
	}
	replay.Active = false
	if sameRecurringEdit(base, replay) {
		t.Fatal("pausing is a distinct edit")
	}
	replay = base
	replay.Input.DropPlaceID = "c"
	if sameRecurringEdit(base, replay) {
		t.Fatal("a different destination is a distinct edit")
	}
}
