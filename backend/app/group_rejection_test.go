package server

import (
	"testing"
	"time"
)

func TestGroupRejectionRequiresFutureUnstartedGroup(t *testing.T) {
	now := time.Now().UTC()
	group := GroupRecord{End: now.Add(time.Hour), Members: []GroupMemberRecord{{UserID: "a"}, {UserID: "b"}}}
	for _, status := range []string{"pending", "confirming", "active", "in_progress", "completed", "cancelled", "expired"} {
		group.Status = status
		want := status == "pending" || status == "confirming" || status == "active"
		if canRejectGroup(group, now) != want {
			t.Fatalf("unexpected rejection eligibility for %s", status)
		}
	}
	group.Status = "confirming"
	group.End = now
	if canRejectGroup(group, now) {
		t.Fatal("expired window was restartable")
	}
	group.End = now.Add(time.Hour)
	group.Members = group.Members[:1]
	if canRejectGroup(group, now) {
		t.Fatal("waiting singleton was rejectable")
	}
}
func TestOccupationIsNotAnAuthorizationRole(t *testing.T) {
	for _, value := range []string{"student", "teacher", "other"} {
		if !validOccupation(value) {
			t.Fatalf("valid occupation %s rejected", value)
		}
	}
	for _, value := range []string{"admin", "rider", "moderator", "", "Teacher"} {
		if validOccupation(value) {
			t.Fatalf("invalid occupation %s accepted", value)
		}
	}
}
