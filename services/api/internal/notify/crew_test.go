package notify

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeTripSource struct{ members []model.Member }

func (f fakeTripSource) Get(context.Context, string) (model.Trip, error) {
	return model.Trip{ID: "t1", Title: "Hanoi food week"}, nil
}
func (f fakeTripSource) Members(context.Context, string) ([]model.Member, error) {
	return f.members, nil
}

type fakePrefs map[string]map[string]bool

func (f fakePrefs) Prefs(_ context.Context, uid string) (map[string]bool, error) { return f[uid], nil }

type recorder struct{ sent map[string][]Message }

func (r *recorder) Send(_ context.Context, d model.Device, m Message) error {
	r.sent[d.ID] = append(r.sent[d.ID], m)
	return nil
}

func rig(prefs fakePrefs) (*Crew, *recorder, *time.Time) {
	members := []model.Member{
		{UID: "asha", DisplayName: "Asha", Status: model.MemberActive},
		{UID: "ben", DisplayName: "Ben", Status: model.MemberActive},
		{UID: "cy", DisplayName: "Cy", Status: model.MemberActive},
		{UID: "gone", DisplayName: "Gone", Status: model.MemberLeft},
	}
	rec := &recorder{sent: map[string][]Message{}}
	clock := time.Now()
	c := &Crew{
		Trips: fakeTripSource{members}, Prefs: prefs, Logger: slog.New(slog.DiscardHandler),
		Push: &Service{Devices: &userDevices{}, Sender: rec},
		now:  func() time.Time { return clock },
	}
	return c, rec, &clock
}

// userDevices gives every person one device named after them.
type userDevices struct{ memDevices }

func (userDevices) List(_ context.Context, uid string) ([]model.Device, error) {
	return []model.Device{{ID: uid}}, nil
}

func TestCrewFanOut(t *testing.T) {
	c, rec, _ := rig(fakePrefs{"cy": {string(KindStopAdded): false}})
	ev := Event{Kind: KindStopAdded, TripID: "t1", ActorUID: "asha"}
	if err := c.Deliver(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	// Not the actor, not someone who left, not someone who turned this kind off.
	if len(rec.sent) != 1 || len(rec.sent["ben"]) != 1 {
		t.Fatalf("sent to %v", rec.sent)
	}
	m := rec.sent["ben"][0]
	if m.Title != "Hanoi food week" || m.Body != "Asha added a place" || m.Data["route"] != "/trips/t1/itinerary" || m.Data["kind"] != "stopAdded" {
		t.Fatalf("message = %+v", m)
	}
}

func TestCrewMuteAllAndQuietWindow(t *testing.T) {
	c, rec, clock := rig(fakePrefs{"ben": {"all": false}})
	ev := Event{Kind: KindExpenseAdded, TripID: "t1", ActorUID: "asha"}
	for range 3 { // a burst of edits is one push
		_ = c.Deliver(context.Background(), ev)
	}
	if len(rec.sent["ben"]) != 0 || len(rec.sent["cy"]) != 1 {
		t.Fatalf("burst: %v", rec.sent)
	}
	*clock = clock.Add(2 * time.Minute) // after the window, the next edit buzzes again
	_ = c.Deliver(context.Background(), ev)
	if len(rec.sent["cy"]) != 2 {
		t.Fatalf("after window: %v", rec.sent)
	}
	// A different kind is not held back by the first.
	_ = c.Deliver(context.Background(), Event{Kind: KindPollCreated, TripID: "t1", ActorUID: "asha"})
	if len(rec.sent["cy"]) != 3 {
		t.Fatalf("other kind: %v", rec.sent)
	}
}

func TestRenderRoutes(t *testing.T) {
	for k, want := range map[Kind]string{
		KindMemberJoined: "/trips/t/crew", KindPollCreated: "/trips/t/polls",
		KindExpenseAdded: "/trips/t/money", KindMeetPoint: "/trips/t/live",
	} {
		if got := Render(Event{Kind: k, TripID: "t"}, "A", "T").Data["route"]; got != want {
			t.Errorf("%s -> %s, want %s", k, got, want)
		}
	}
}
