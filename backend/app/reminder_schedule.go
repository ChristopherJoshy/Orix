package server

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MemberReminder is private persisted scheduling data. Group responses project
// only the requesting member's pickup time/label, never this whole array.
type MemberReminder struct {
	UserID   string    `bson:"user_id" json:"-"`
	RideTime time.Time `bson:"ride_time" json:"-"`
	Pickup   string    `bson:"pickup_summary" json:"-"`
}

func roadPlanHasOffsets(plan *RoadPlan) bool {
	if plan == nil || len(plan.Stops) < 2 || len(plan.Stops) > 8 || len(plan.Stops)%2 != 0 {
		return false
	}
	previous := 0.0
	for i, stop := range plan.Stops {
		if stop.OffsetSeconds == nil || !finitePositive(*stop.OffsetSeconds) || *stop.OffsetSeconds > plan.Duration+1 || *stop.OffsetSeconds < previous {
			return false
		}
		if i == 0 && *stop.OffsetSeconds != 0 {
			return false
		}
		previous = *stop.OffsetSeconds
	}
	return finitePositive(plan.Duration)
}

// BuildMemberReminders uses validated road leg offsets, not a literal shared
// departure for every lift passenger. Route/traffic durations remain estimates.
func BuildMemberReminders(group GroupRecord, pickupLabels map[string]string) ([]MemberReminder, error) {
	if group.Departure.IsZero() || len(group.Members) < 2 {
		return nil, ErrGroupUnavailable
	}
	offsets := map[string]float64{}
	if group.Mode == "lift" {
		if !roadPlanHasOffsets(group.RoadPlan) || len(group.RoadPlan.Stops) != 2*len(group.Members) {
			return nil, ErrNoRoadPlan
		}
		for _, stop := range group.RoadPlan.Stops {
			if stop.Kind != "pickup" {
				continue
			}
			if _, exists := offsets[stop.UserID]; exists {
				return nil, ErrNoRoadPlan
			}
			offsets[stop.UserID] = *stop.OffsetSeconds
		}
		if len(offsets) != len(group.Members) {
			return nil, ErrNoRoadPlan
		}
	}
	result := make([]MemberReminder, 0, len(group.Members))
	seen := map[string]bool{}
	for _, member := range group.Members {
		if member.UserID == "" || seen[member.UserID] {
			return nil, ErrGroupUnavailable
		}
		seen[member.UserID] = true
		when, pickup := group.Departure, group.Pickup
		if group.Mode == "lift" {
			offset, exists := offsets[member.UserID]
			if !exists || pickupLabels[member.UserID] == "" {
				return nil, ErrNoRoadPlan
			}
			when = when.Add(time.Duration(offset * float64(time.Second)))
			pickup = pickupLabels[member.UserID]
		}
		// Mongo timestamps have millisecond precision. Persist the exact value used
		// for subsequent outbox identity and stale-schedule comparisons.
		result = append(result, MemberReminder{member.UserID, when.UTC().Truncate(time.Millisecond), pickup})
	}
	return result, nil
}
func (m mongoUsers) prepareMemberReminders(ctx context.Context, group GroupRecord) ([]MemberReminder, error) {
	labels := map[string]string{}
	if group.Mode == "lift" {
		cursor, err := m.rides().Find(ctx, bson.M{"group_id": group.ID, "status": "matched"}, options.Find().SetLimit(5).SetProjection(bson.M{"user_id": 1, "pickup_label": 1}))
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var ride Ride
			if err = cursor.Decode(&ride); err != nil {
				return nil, err
			}
			if _, exists := labels[ride.UserID]; exists {
				return nil, ErrNoRoadPlan
			}
			labels[ride.UserID] = ride.PickupLabel
		}
		if err := cursor.Err(); err != nil {
			return nil, err
		}
		if len(labels) != len(group.Members) {
			return nil, ErrNoRoadPlan
		}
	}
	return BuildMemberReminders(group, labels)
}

func reminderForMember(group GroupRecord, user string) (MemberReminder, bool) {
	if len(group.Members) < 2 {
		return MemberReminder{}, false
	}
	if group.Status != "active" && !(group.Status == "in_progress" && group.Mode == "lift") {
		return MemberReminder{}, false
	}
	present := false
	for _, member := range group.Members {
		if member.UserID == user {
			present = !member.Finished
			break
		}
	}
	if !present {
		return MemberReminder{}, false
	}
	return memberPlannedPickup(group, user)
}
func memberPlannedPickup(group GroupRecord, user string) (MemberReminder, bool) {
	for _, planned := range group.Reminders {
		if planned.UserID == user {
			return planned, !planned.RideTime.IsZero()
		}
	}
	// Legacy shared-auto/partner groups can use their common departure. Never
	// invent a lift pickup time from the driver's departure when offsets are absent.
	if len(group.Reminders) == 0 && group.Mode != "lift" && !group.Departure.IsZero() {
		return MemberReminder{user, group.Departure, group.Pickup}, true
	}
	return MemberReminder{}, false
}
func matchingReminder(group GroupRecord, event deliveryEvent, now time.Time) (MemberReminder, bool) {
	planned, ok := reminderForMember(group, event.Target)
	if !ok || !group.Departure.Equal(event.Departure) || !planned.RideTime.After(now.Add(10*time.Second)) {
		return MemberReminder{}, false
	}
	expected := event.RideTime
	if expected.IsZero() && group.Mode != "lift" {
		expected = event.Departure
	}
	return planned, planned.RideTime.Equal(expected)
}

func reminderEventID(group string, departure, rideTime time.Time, user string) string {
	return fmt.Sprintf("ride15v2:%s:%d:%d:%s", group, departure.UnixMilli(), rideTime.UnixMilli(), stableID(user))
}
