package analytics

import (
	"testing"
	"time"
)

func TestClean(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	raw := []Raw{
		{Name: "trip_created", At: now, Props: map[string]any{"destinations": 2.0, "solo": true, "source": "onboarding"}},
		{Name: "tapped_secret_thing", At: now},                    // not on the list
		{Name: "sign_in", At: now.Add(-8 * 24 * time.Hour)},       // too old
		{Name: "sign_in", At: now.Add(time.Hour)},                 // from the future
		{Name: "paywall_shown", At: now.Add(-3 * 24 * time.Hour)}, // held offline, fine
		{Name: "screen_viewed", At: now, Props: map[string]any{
			"phone":     "+14155550123",              // banned key
			"userEmail": "a@b.c",                     // banned key
			"title":     "My trip to Hanoi with Sam", // spaces: free text
			"screen":    "trip home",                 // spaces
			"route":     "trip.home",
			"lat":       21.0,
			"nested":    map[string]any{"a": 1.0},
			"Bad Key":   "x",
		}},
	}
	kept, dropped := Clean(raw, now)
	if dropped != 3 || len(kept) != 3 {
		t.Fatalf("kept %d dropped %d", len(kept), dropped)
	}
	if p := kept[0].Props; len(p) != 3 || p["destinations"] != 2.0 || p["solo"] != true || p["source"] != "onboarding" {
		t.Errorf("props kept: %v", p)
	}
	if p := kept[2].Props; len(p) != 1 || p["route"] != "trip.home" {
		t.Errorf("only the identifier survives: %v", p)
	}
}

func TestTooManyProps(t *testing.T) {
	props := map[string]any{}
	for _, k := range []string{"a1", "b1", "c1", "d1", "e1", "f1", "g1", "h1", "i1", "j1", "k1", "l1"} {
		props[k] = true
	}
	kept, _ := Clean([]Raw{{Name: "sign_in", At: time.Now(), Props: props}}, time.Now())
	if len(kept[0].Props) != MaxProps {
		t.Errorf("props = %d, want %d", len(kept[0].Props), MaxProps)
	}
}
