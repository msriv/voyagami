package firestorerepo

import (
	"errors"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func newStop(tripID, date, order string) model.Stop {
	return model.Stop{
		ID: model.NewID(), TripID: tripID, Date: date, Order: order, TimeZone: "Asia/Ho_Chi_Minh",
		Place: model.PlaceRef{Name: "Train Street", Lat: 21.0245, Lng: 105.8412},
	}
}

func TestStopsLifecycle(t *testing.T) {
	client := testClient(t)
	trips, stops := NewTrips(client), NewStops(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")

	a, err := stops.Create(ctx, newStop(trip.ID, "2026-11-22", "V"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	withInfo := newStop(trip.ID, "2026-11-22", "k")
	withInfo.Place.Address, withInfo.Place.Category = "Phung Hung, Hanoi", "sights"
	if got, err := stops.Create(ctx, withInfo); err != nil || got.Place.Address != "Phung Hung, Hanoi" || got.Place.Category != "sights" {
		t.Fatalf("address and category: %+v, %v", got, err)
	}
	if _, err := stops.Update(ctx, trip.ID, withInfo.ID, model.StopPatch{Deleted: true}); err != nil {
		t.Fatal(err)
	}
	if a.Place.Name != "Train Street" || !a.Place.Custom() || a.CreatedAt.IsZero() {
		t.Errorf("stop = %+v", a)
	}
	b, _ := stops.Create(ctx, newStop(trip.ID, "2026-11-22", "G"))
	c, _ := stops.Create(ctx, newStop(trip.ID, "2026-11-21", "Z"))

	list, err := stops.List(ctx, trip.ID)
	if err != nil || len(list) != 3 || list[0].ID != c.ID || list[1].ID != b.ID || list[2].ID != a.ID {
		t.Fatalf("list = %+v, %v", list, err)
	}

	// Move a to the other day, set and then clear time fields.
	day, ord, start, dur := "2026-11-21", "a", "09:30", 90
	got, err := stops.Update(ctx, trip.ID, a.ID, model.StopPatch{Date: &day, Order: &ord, StartLocal: &start, DurationMin: &dur})
	if err != nil || got.Date != day || got.Order != "a" || got.StartLocal != "09:30" || got.DurationMin != 90 {
		t.Fatalf("update = %+v, %v", got, err)
	}
	empty, zero := "", 0
	got, err = stops.Update(ctx, trip.ID, a.ID, model.StopPatch{StartLocal: &empty, DurationMin: &zero})
	if err != nil || got.StartLocal != "" || got.DurationMin != 0 {
		t.Fatalf("clear = %+v, %v", got, err)
	}

	if _, err := stops.Update(ctx, trip.ID, b.ID, model.StopPatch{Deleted: true}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = stops.List(ctx, trip.ID)
	if len(list) != 2 {
		t.Errorf("deleted stop should be hidden, got %d", len(list))
	}
	if kept, err := stops.Get(ctx, trip.ID, b.ID); err != nil || kept.DeletedAt == nil {
		t.Errorf("tombstone should stay readable: %+v, %v", kept, err)
	}

	if _, err := stops.Get(ctx, trip.ID, "nope"); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("missing stop: %v", err)
	}
	if _, err := stops.Update(ctx, trip.ID, "nope", model.StopPatch{Deleted: true}); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("update missing: %v", err)
	}
	bad := newStop(trip.ID, "22/11", "V")
	if _, err := stops.Create(ctx, bad); !model.IsValidation(err) {
		t.Errorf("bad date should be a validation error: %v", err)
	}
}

func TestStopsSetOrders(t *testing.T) {
	client := testClient(t)
	trips, stops := NewTrips(client), NewStops(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")
	a, _ := stops.Create(ctx, newStop(trip.ID, "2026-11-22", "a"))
	b, _ := stops.Create(ctx, newStop(trip.ID, "2026-11-22", "b"))

	if err := stops.SetOrders(ctx, trip.ID, map[string]string{a.ID: "k", b.ID: "c"}); err != nil {
		t.Fatalf("SetOrders: %v", err)
	}
	list, _ := stops.List(ctx, trip.ID)
	if len(list) != 2 || list[0].ID != b.ID || list[1].ID != a.ID {
		t.Fatalf("list = %+v", list)
	}

	// All or nothing: one missing stop leaves the others as they were.
	err := stops.SetOrders(ctx, trip.ID, map[string]string{a.ID: "z", "nope": "y"})
	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("missing stop: %v", err)
	}
	if got, _ := stops.Get(ctx, trip.ID, a.ID); got.Order != "k" {
		t.Errorf("partial write: order = %q", got.Order)
	}
}
