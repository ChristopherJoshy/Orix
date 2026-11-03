package server

import "testing"

func TestResourceInvalidationsAreExplicit(t *testing.T) {
	for _, test := range []struct{ collection, event string }{
		{"ride_requests", "ride_update"}, {"notifications", "notification_update"}, {"support_tickets", "support_update"},
		{"users", ""}, {"reports", ""}, {"notification_outbox", ""}, {"unknown", ""},
	} {
		if got := resourceEventType(test.collection); got != test.event {
			t.Errorf("%s: %q, want %q", test.collection, got, test.event)
		}
	}
}
