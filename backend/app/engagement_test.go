package server

import (
	"testing"
	"time"
)

func TestEngagementScheduleBoundaries(t *testing.T) {
	ist := time.FixedZone("IST", 19800)
	for _, test := range []struct {
		hour, minute int
		want         bool
	}{{7, 59, false}, {8, 0, true}, {8, 29, true}, {8, 30, false}, {9, 0, false}, {12, 0, true}, {16, 15, true}, {20, 29, true}, {20, 30, false}, {21, 0, false}, {0, 0, false}} {
		now := time.Date(2026, 11, 3, test.hour, test.minute, 0, 0, ist)
		start, end, ok := engagementSlot(now.UTC())
		if ok != test.want {
			t.Errorf("%02d:%02d got %v", test.hour, test.minute, ok)
		}
		if ok && (end.Sub(start) != 30*time.Minute || start.Location() != time.UTC || now.Before(start) || !now.Before(end)) {
			t.Fatal("invalid UTC window")
		}
	}
}
func TestEngagementContentAndRotation(t *testing.T) {
	config := defaultEngagement()
	if len(config.Messages) != 20 || !validateEngagement(config) {
		t.Fatal("expected twenty valid authored messages")
	}
	slot := time.Date(2026, 11, 3, 2, 30, 0, 0, time.UTC)
	first, ok := chooseEngagement(config, slot, "layout-test-person")
	repeat, _ := chooseEngagement(config, slot, "layout-test-person")
	next, _ := chooseEngagement(config, slot.Add(4*time.Hour), "layout-test-person")
	if !ok || first.ID != repeat.ID || first.ID == next.ID {
		t.Fatal("selection must be stable within a slot and rotate afterward")
	}
	for i := range config.Messages {
		config.Messages[i].Enabled = false
	}
	if _, ok := chooseEngagement(config, slot, "layout-test-person"); ok {
		t.Fatal("paused messages must not be selected")
	}
	config = defaultEngagement()
	config.Enabled = false
	if _, ok := chooseEngagement(config, slot, "layout-test-person"); ok {
		t.Fatal("disabled schedule must not select a message")
	}
}
func TestEngagementRejectsInvalidEdits(t *testing.T) {
	tests := []struct {
		name string
		edit func(*EngagementConfig)
	}{
		{"duplicate", func(c *EngagementConfig) { c.Messages[1].ID = c.Messages[0].ID }},
		{"empty", func(c *EngagementConfig) { c.Messages[0].Body = " " }},
		{"newline", func(c *EngagementConfig) { c.Messages[0].Title = "hello\nthere" }},
		{"bad ID", func(c *EngagementConfig) { c.Messages[0].ID = "a.b" }},
		{"missing version", func(c *EngagementConfig) { c.Version = 0 }},
		{"too long", func(c *EngagementConfig) {
			for range 241 {
				c.Messages[0].Body += "a"
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := defaultEngagement()
			test.edit(&c)
			if validateEngagement(c) {
				t.Fatal("accepted invalid edit")
			}
		})
	}
}
