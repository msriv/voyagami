package main

import (
	"testing"
	"time"
)

func TestPassCovers(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		end  string
		want bool
	}{
		{"trip still on", "2026-11-12", true},
		{"last day", "2026-11-10", true},
		{"trip ends in two days", "2026-11-12", true},
		{"over by more than the grace", "2026-11-06", false},
		{"no end date keeps the pass", "", true},
		{"unreadable date keeps the pass", "soon", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := passCovers(tt.end, now); got != tt.want {
				t.Errorf("passCovers(%q) = %v, want %v", tt.end, got, tt.want)
			}
		})
	}
	// Edge: the pass ends at the start of the third day after the trip.
	end := "2026-11-10"
	if !passCovers(end, time.Date(2026, 11, 12, 23, 59, 0, 0, time.UTC)) || passCovers(end, time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)) {
		t.Error("pass must end at midnight UTC on the third day after the last day")
	}
}
