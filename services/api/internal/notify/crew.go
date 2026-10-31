package notify

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

// Kind is a kind of crew event a person can be told about.
type Kind string

// The crew events that send a push.
const (
	KindMemberJoined Kind = "memberJoined"
	KindStopAdded    Kind = "stopAdded"
	KindPollCreated  Kind = "pollCreated"
	KindExpenseAdded Kind = "expenseAdded"
	KindMeetPoint    Kind = "meetPoint"
)

// Event is something that happened on a trip. It holds ids, not content.
type Event struct {
	Kind     Kind
	TripID   string
	ActorUID string
}

// TripSource is what the fan-out reads about a trip.
type TripSource interface {
	Get(ctx context.Context, id string) (model.Trip, error)
	Members(ctx context.Context, tripID string) ([]model.Member, error)
}

// PrefSource returns a person's notification settings: kind to on or off.
type PrefSource interface {
	Prefs(ctx context.Context, uid string) (map[string]bool, error)
}

// Crew turns trip events into pushes for the other members. It respects each
// person's settings and sends at most one push per kind, trip, actor and
// person in a minute, so a burst of edits is one buzz, not ten.
//
// shortcut: the quiet window is in memory per instance and delivery runs in
// the request's process, not off a Pub/Sub queue; move both when the API
// scales past a few instances or pushes must survive a restart (P1).
type Crew struct {
	Trips  TripSource
	Prefs  PrefSource
	Push   *Service
	Logger *slog.Logger
	// Window is the quiet time per kind, trip, actor and person. Zero means a minute.
	Window time.Duration

	mu   sync.Mutex
	seen map[string]time.Time
	now  func() time.Time
}

// Publish delivers the event without holding up the caller. Errors are logged.
func (c *Crew) Publish(_ context.Context, ev Event) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := c.Deliver(ctx, ev); err != nil {
			c.Logger.Warn("crew push failed", "kind", ev.Kind, "error", err)
		}
	}()
}

// Deliver sends the event's push to every other active member who wants it.
func (c *Crew) Deliver(ctx context.Context, ev Event) error {
	trip, err := c.Trips.Get(ctx, ev.TripID)
	if err != nil {
		return err
	}
	members, err := c.Trips.Members(ctx, ev.TripID)
	if err != nil {
		return err
	}
	actor := "Someone"
	for _, m := range members {
		if m.UID == ev.ActorUID && m.DisplayName != "" {
			actor = m.DisplayName
		}
	}
	msg := Render(ev, actor, trip.Title)
	var first error
	for _, m := range members {
		if m.UID == ev.ActorUID || m.Status != model.MemberActive {
			continue
		}
		prefs, err := c.Prefs.Prefs(ctx, m.UID)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		if !wants(prefs, ev.Kind) || !c.fresh(ev, m.UID) {
			continue
		}
		if _, err := c.Push.SendToUser(ctx, m.UID, msg); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// wants is true unless the person turned this kind, or all pushes, off.
func wants(prefs map[string]bool, k Kind) bool {
	if on, ok := prefs["all"]; ok && !on {
		return false
	}
	if on, ok := prefs[string(k)]; ok && !on {
		return false
	}
	return true
}

// fresh reports whether a push for this event may go out now, and marks it sent.
func (c *Crew) fresh(ev Event, uid string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now
	if c.now != nil {
		now = c.now
	}
	window := c.Window
	if window == 0 {
		window = time.Minute
	}
	if c.seen == nil {
		c.seen = map[string]time.Time{}
	}
	key := fmt.Sprintf("%s|%s|%s|%s", ev.Kind, ev.TripID, ev.ActorUID, uid)
	t := now()
	if last, ok := c.seen[key]; ok && t.Sub(last) < window {
		return false
	}
	c.seen[key] = t
	if len(c.seen) > 5000 { // forget old keys
		for k, v := range c.seen {
			if t.Sub(v) > window {
				delete(c.seen, k)
			}
		}
	}
	return true
}

// Render writes the push for an event. The text uses only names the person
// already sees in the app: the actor's name and the trip's title. The data
// carries the deep link route.
func Render(ev Event, actor, tripTitle string) Message {
	var body, screen string
	switch ev.Kind {
	case KindMemberJoined:
		body, screen = actor+" joined the trip", "crew"
	case KindStopAdded:
		body, screen = actor+" added a place", "itinerary"
	case KindPollCreated:
		body, screen = actor+" started a poll", "polls"
	case KindExpenseAdded:
		body, screen = actor+" added an expense", "money"
	case KindMeetPoint:
		body, screen = actor+" suggested a meeting spot", "live"
	default:
		body, screen = actor+" made a change", ""
	}
	route := "/trips/" + ev.TripID
	if screen != "" {
		route += "/" + screen
	}
	return Message{
		Title: tripTitle,
		Body:  body,
		Data:  map[string]string{"kind": string(ev.Kind), "tripId": ev.TripID, "route": route},
	}
}
