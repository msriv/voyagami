// Package analytics checks product events before they are stored. The rules keep
// phone numbers, places and free text out of them: names come from a fixed list
// and property values are short identifiers, numbers or flags.
package analytics

import (
	"math"
	"regexp"
	"strings"
	"time"
)

// Names are the events the app may send. Add a name here and in docs/analytics.md
// together.
var Names = map[string]bool{
	"app_installed":        true,
	"sign_in":              true,
	"onboarding_completed": true,
	"trip_created":         true,
	"invite_sent":          true,
	"crew_joined":          true,
	"place_added":          true,
	"expense_added":        true,
	"journal_entry_added":  true,
	"ask_draft_created":    true,
	"trial_started":        true,
	"subscription_started": true,
	"trip_pass_bought":     true,
	"paywall_shown":        true,
	"paywall_dismissed":    true,
	"paywall_converted":    true,
	"screen_viewed":        true,
}

// Limits on one event.
const (
	MaxProps    = 10
	MaxValueLen = 64
	maxAge      = 7 * 24 * time.Hour // the phone may hold events while offline
	maxFuture   = 5 * time.Minute
)

var (
	keyRe   = regexp.MustCompile(`^[a-z][a-zA-Z0-9]{0,31}$`)
	valueRe = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)
	// Property names that suggest personal data are refused whatever the value.
	banned = []string{"phone", "email", "name", "address", "lat", "lng", "lon", "location", "text", "note", "message", "token"}
)

// Event is one checked event.
type Event struct {
	Name  string
	At    time.Time
	Props map[string]any // string, float64 or bool
}

// Raw is an event as the phone sent it.
type Raw struct {
	Name  string
	At    time.Time
	Props map[string]any
}

// Clean keeps the events that pass and cleans their properties. It returns the
// events kept and how many were dropped whole.
func Clean(raw []Raw, now time.Time) (kept []Event, dropped int) {
	for _, r := range raw {
		if !Names[r.Name] || r.At.Before(now.Add(-maxAge)) || r.At.After(now.Add(maxFuture)) {
			dropped++
			continue
		}
		kept = append(kept, Event{Name: r.Name, At: r.At.UTC(), Props: cleanProps(r.Props)})
	}
	return kept, dropped
}

func cleanProps(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		if len(out) >= MaxProps || !keyRe.MatchString(k) || bannedKey(k) {
			continue
		}
		switch x := v.(type) {
		case string:
			if valueRe.MatchString(x) {
				out[k] = x
			}
		case float64:
			if !math.IsNaN(x) && !math.IsInf(x, 0) {
				out[k] = x
			}
		case bool:
			out[k] = x
		}
	}
	return out
}

func bannedKey(k string) bool {
	l := strings.ToLower(k)
	for _, b := range banned {
		if strings.Contains(l, b) {
			return true
		}
	}
	return false
}
