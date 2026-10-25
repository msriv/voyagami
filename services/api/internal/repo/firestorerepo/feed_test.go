package firestorerepo

import (
	"context"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestFeedReportsChangesAfterStart(t *testing.T) {
	client := testClient(t)
	trips, stops := NewTrips(client), NewStops(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")

	got := make(chan model.Change, 8)
	errc := make(chan error, 1)
	wctx, stop := context.WithCancel(ctx)
	go func() { errc <- NewFeed(client).Watch(wctx, trip.ID, func(c model.Change) { got <- c }) }()
	time.Sleep(time.Second) // let the listeners attach

	s, err := stops.Create(ctx, newStop(trip.ID, "2026-11-22", "V"))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-got:
		if c != (model.Change{Kind: "stops", ID: s.ID, Op: "upsert"}) {
			t.Fatalf("got %+v", c)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no change seen")
	}
	stop()
	if err := <-errc; err != nil {
		t.Fatalf("watch: %v", err)
	}
}
