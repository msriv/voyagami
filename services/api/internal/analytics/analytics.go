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

// Names are the events the app and server may send: the VOY-224 sheet. Add a
// name here, in the sheet and in apps/mobile/src/analytics/events.ts together.
var Names = map[string]bool{
	"app_installed":                true,
	"app_open":                     true,
	"screen_viewed":                true,
	"sign_in_started":              true,
	"sign_in":                      true,
	"sign_out":                     true,
	"onboarding_step_completed":    true,
	"onboarding_completed":         true,
	"trip_created":                 true,
	"trip_opened":                  true,
	"trip_deleted":                 true,
	"offline_plan_saved":           true,
	"sync_failed":                  true,
	"place_added":                  true,
	"place_removed":                true,
	"best_order_applied":           true,
	"itinerary_reordered":          true,
	"map_opened":                   true,
	"booking_added":                true,
	"booking_imported":             true,
	"invite_sent":                  true,
	"crew_joined":                  true,
	"crew_role_changed":            true,
	"poll_created":                 true,
	"poll_voted":                   true,
	"comment_added":                true,
	"live_location_started":        true,
	"expense_added":                true,
	"settle_up_recorded":           true,
	"journal_entry_added":          true,
	"photo_added":                  true,
	"recap_viewed":                 true,
	"recap_shared":                 true,
	"recap_video_rendered":         true,
	"place_searched":               true,
	"recs_shown":                   true,
	"rec_tapped":                   true,
	"sponsored_hidden":             true,
	"ask_draft_created":            true,
	"ask_draft_accepted":           true,
	"paywall_shown":                true,
	"paywall_dismissed":            true,
	"paywall_converted":            true,
	"limit_reached":                true,
	"trial_started":                true,
	"subscription_started":         true,
	"trip_pass_bought":             true,
	"paid":                         true,
	"subscription_renewed":         true,
	"subscription_cancelled":       true,
	"purchase_refunded":            true,
	"push_opened":                  true,
	"notification_setting_changed": true,
	"data_export_requested":        true,
	"account_deleted":              true,
	"account_purged":               true,
	"help_opened":                  true,
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
