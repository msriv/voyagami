package firestorerepo

import (
	"errors"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestMeetPoints(t *testing.T) {
	client := testClient(t)
	trips, meets := NewTrips(client), NewMeetPoints(client)
	ctx := testCtx(t)
	trip, owner := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")
	at := time.Now().Add(time.Hour)

	m := model.MeetPoint{ID: model.NewID(), Name: "Gate", CreatedBy: owner, At: at, Place: model.PlaceRef{Name: "Lake", Lat: 21.03, Lng: 105.85}}
	got, err := meets.Create(ctx, trip.ID, m)
	if err != nil || got.Place.Name != "Lake" || !got.At.Equal(at.UTC().Truncate(time.Microsecond)) || got.CreatedAt.IsZero() || len(got.ConfirmedBy) != 0 {
		t.Fatalf("Create: %+v, %v", got, err)
	}
	if _, err := meets.Create(ctx, trip.ID, m); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate: %v", err)
	}
	if _, err := meets.Create(ctx, trip.ID, model.MeetPoint{ID: "x", CreatedBy: owner}); !model.IsValidation(err) {
		t.Errorf("invalid: %v", err)
	}
	undated := model.MeetPoint{ID: model.NewID(), Name: "Later", CreatedBy: owner, Place: model.PlaceRef{PlaceID: "p1"}}
	if _, err := meets.Create(ctx, trip.ID, undated); err != nil {
		t.Fatal(err)
	}

	for _, coming := range []bool{true, true, false} {
		if got, err = meets.SetConfirmed(ctx, trip.ID, m.ID, "u1", coming); err != nil {
			t.Fatal(err)
		}
	}
	if len(got.ConfirmedBy) != 0 {
		t.Errorf("after toggling: %v", got.ConfirmedBy)
	}
	if got, _ = meets.SetConfirmed(ctx, trip.ID, m.ID, "u2", true); len(got.ConfirmedBy) != 1 {
		t.Errorf("confirmed: %v", got.ConfirmedBy)
	}
	if _, err := meets.SetConfirmed(ctx, trip.ID, "missing", "u1", true); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("missing: %v", err)
	}

	name := "North gate"
	if got, err = meets.Update(ctx, trip.ID, m.ID, model.MeetPointPatch{Name: &name}); err != nil || got.Name != name || got.Place.Name != "Lake" {
		t.Errorf("Update: %+v, %v", got, err)
	}
	list, err := meets.List(ctx, trip.ID)
	if err != nil || len(list) != 2 || list[0].ID != m.ID || list[1].ID != undated.ID {
		t.Fatalf("List: %+v, %v", list, err)
	}
	if err := meets.Delete(ctx, trip.ID, m.ID); err != nil {
		t.Fatal(err)
	}
	if err := meets.Delete(ctx, trip.ID, m.ID); err != nil {
		t.Errorf("delete twice: %v", err)
	}
}

func TestPresenceExpires(t *testing.T) {
	client := testClient(t)
	trips, pres := NewTrips(client), NewPresence(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")
	now := time.Now()

	live := model.Presence{UID: "u1", Lat: 21.03, Lng: 105.85, AccuracyM: 10, UpdatedAt: now, EndsAt: now.Add(time.Hour)}
	over := model.Presence{UID: "u2", Lat: 21.04, Lng: 105.86, UpdatedAt: now.Add(-2 * time.Hour), EndsAt: now.Add(-time.Hour)}
	for _, p := range []model.Presence{live, over} {
		if err := pres.Put(ctx, trip.ID, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := pres.Put(ctx, trip.ID, model.Presence{UID: "u3", Lat: 200, UpdatedAt: now, EndsAt: now.Add(time.Hour)}); !model.IsValidation(err) {
		t.Errorf("bad coordinates: %v", err)
	}
	if err := pres.Put(ctx, trip.ID, model.Presence{UID: "u3", UpdatedAt: now, EndsAt: now}); !model.IsValidation(err) {
		t.Errorf("already over: %v", err)
	}
	got, err := pres.List(ctx, trip.ID, now)
	if err != nil || len(got) != 1 || got[0].UID != "u1" || got[0].AccuracyM != 10 {
		t.Fatalf("List: %+v, %v", got, err)
	}
	// The ended entry is removed, not just hidden.
	if _, err := pres.Get(ctx, trip.ID, "u2"); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("ended entry still stored: %v", err)
	}
	if p, err := pres.Get(ctx, trip.ID, "u1"); err != nil || !p.EndsAt.Equal(live.EndsAt.UTC().Truncate(time.Microsecond)) {
		t.Errorf("Get: %+v, %v", p, err)
	}
	if err := pres.Delete(ctx, trip.ID, "u1"); err != nil {
		t.Fatal(err)
	}
	if err := pres.Delete(ctx, trip.ID, "u1"); err != nil {
		t.Errorf("delete twice: %v", err)
	}
}
