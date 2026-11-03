package server

import (
	"testing"
	"time"
)

func scheduledLift(t *testing.T) GroupRecord {
	t.Helper()
	start := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
	seconds := func(value float64) *float64 { return &value }
	group := GroupRecord{
		ID: "lift", Mode: "lift", Status: "active", Departure: start, Pickup: "Driver origin",
		Members: []GroupMemberRecord{{UserID: "driver", Role: "rider", Ready: true}, {UserID: "passenger", Role: "passenger", Ready: true}},
		RoadPlan: &RoadPlan{Duration: 3600, Stops: []RoadStop{
			{UserID: "driver", Kind: "pickup", OffsetSeconds: seconds(0)},
			{UserID: "passenger", Kind: "pickup", OffsetSeconds: seconds(1800)},
			{UserID: "passenger", Kind: "dropoff", OffsetSeconds: seconds(3000)},
			{UserID: "driver", Kind: "dropoff", OffsetSeconds: seconds(3600)},
		}},
	}
	var err error
	group.Reminders, err = BuildMemberReminders(group, map[string]string{"driver": "Driver origin", "passenger": "Passenger stop"})
	if err != nil {
		t.Fatal(err)
	}
	return group
}
func TestLiftRemindersUsePersonalPickupAfterDriverStarts(t *testing.T) {
	group := scheduledLift(t)
	group.Status = "in_progress"
	now := group.Departure.Add(15 * time.Minute)
	event := deliveryEvent{Target: "passenger", Departure: group.Departure, RideTime: group.Departure.Add(30 * time.Minute)}
	planned, ok := matchingReminder(group, event, now)
	if !ok || planned.RideTime.Sub(now) != 15*time.Minute || planned.Pickup != "Passenger stop" {
		t.Fatal("later pickup did not remain eligible", planned, ok)
	}
	driver := event
	driver.Target = "driver"
	driver.RideTime = group.Departure
	if _, ok := matchingReminder(group, driver, now); ok {
		t.Fatal("driver received a past-departure alarm")
	}
}
func TestReminderRejectsChangedOrClosedCoordination(t *testing.T) {
	original := scheduledLift(t)
	event := deliveryEvent{Target: "passenger", Departure: original.Departure, RideTime: original.Reminders[1].RideTime}
	now := original.Departure.Add(15 * time.Minute)
	cases := []struct {
		name   string
		change func(*GroupRecord)
	}{
		{"cancelled", func(g *GroupRecord) { g.Status = "cancelled" }},
		{"expired", func(g *GroupRecord) { g.Status = "expired" }},
		{"completed", func(g *GroupRecord) { g.Status = "completed" }},
		{"changed driver departure", func(g *GroupRecord) { g.Departure = g.Departure.Add(time.Minute) }},
		{"changed pickup", func(g *GroupRecord) { g.Reminders[1].RideTime = g.Reminders[1].RideTime.Add(time.Minute) }},
		{"member finished", func(g *GroupRecord) { g.Members[1].Finished = true }},
		{"member removed", func(g *GroupRecord) { g.Members = g.Members[:1] }},
		{"schedule cleared", func(g *GroupRecord) { g.Reminders = nil }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			group := original
			group.Members = append([]GroupMemberRecord(nil), original.Members...)
			group.Reminders = append([]MemberReminder(nil), original.Reminders...)
			test.change(&group)
			if _, ok := matchingReminder(group, event, now); ok {
				t.Fatal("stale reminder remained eligible")
			}
		})
	}
	if _, ok := matchingReminder(original, event, event.RideTime.Add(-5*time.Second)); ok {
		t.Fatal("near-pickup reminder should be skipped")
	}
}
func TestPersonalPickupRequiresVerifiedOffsetsAndLabels(t *testing.T) {
	original := scheduledLift(t)
	missing := original
	missing.RoadPlan = &RoadPlan{Duration: original.RoadPlan.Duration, Stops: append([]RoadStop(nil), original.RoadPlan.Stops...)}
	missing.RoadPlan.Stops[1].OffsetSeconds = nil
	labels := map[string]string{"driver": "Driver origin", "passenger": "Passenger stop"}
	if _, err := BuildMemberReminders(missing, labels); err == nil {
		t.Fatal("unknown offset was guessed")
	}
	if _, err := BuildMemberReminders(original, nil); err == nil {
		t.Fatal("unknown pickup label was guessed")
	}
	shared := original
	shared.Mode = "auto"
	shared.RoadPlan = nil
	planned, err := BuildMemberReminders(shared, nil)
	if err != nil || !planned[0].RideTime.Equal(shared.Departure) || !planned[1].RideTime.Equal(shared.Departure) {
		t.Fatal("shared trip should retain common departure", err)
	}
}
func TestReminderIdentityIncludesBothSchedules(t *testing.T) {
	group := scheduledLift(t)
	pickup := group.Reminders[1].RideTime
	id := reminderEventID(group.ID, group.Departure, pickup, "passenger")
	if id != reminderEventID(group.ID, group.Departure, pickup, "passenger") {
		t.Fatal("retry identity changed")
	}
	if id == reminderEventID(group.ID, group.Departure.Add(time.Millisecond), pickup, "passenger") {
		t.Fatal("driver reschedule did not change identity")
	}
	if id == reminderEventID(group.ID, group.Departure, pickup.Add(time.Millisecond), "passenger") {
		t.Fatal("personal reschedule did not change identity")
	}
	if id == reminderEventID(group.ID, group.Departure, pickup, "driver") {
		t.Fatal("recipients shared identity")
	}
}
