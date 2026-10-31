package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/importer"
	"github.com/msriv/voyagami/services/api/internal/model"
)

// livePlans reads the counters the handlers write, like the real plan source.
type livePlans struct {
	plus     bool
	upcoming int
	usage    map[model.UsageCounter]int
}

func (l *livePlans) User(context.Context, string) (entitlements.User, error) {
	return entitlements.User{Plus: l.plus, Used: map[entitlements.Feature]int{entitlements.UpcomingTrips: l.upcoming}}, nil
}

func (l *livePlans) Trip(context.Context, string) (entitlements.Trip, error) {
	return entitlements.Trip{Used: map[entitlements.Feature]int{
		entitlements.BestOrder:         l.usage[model.UsageBestOrderRuns],
		entitlements.ForwardedBookings: l.usage[model.UsageForwardedBookings],
	}}, nil
}

func (l *livePlans) Add(_ context.Context, _ string, c model.UsageCounter, n int) error {
	l.usage[c] += n
	return nil
}

type oneBooking struct{}

func (oneBooking) Parse(context.Context, importer.Input) ([]importer.Candidate, error) {
	return []importer.Candidate{{Booking: model.Booking{Kind: "flight", Title: "VN 1"}, Confidence: 0.9}}, nil
}

func TestPlanLimits(t *testing.T) {
	plans := &livePlans{usage: map[model.UsageCounter]int{}}
	trips := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips),
		WithStops(&fakeStops{stops: map[string]model.Stop{}}),
		WithBookings(&fakeBookings{byID: map[string]model.Booking{}}), WithImporter(oneBooking{}),
		WithUsage(plans), WithEntitlements(entitlements.New(entitlements.Free()), plans))

	// Trips: the third upcoming trip is refused with limit_reached and the feature.
	plans.upcoming = 1
	if rec := call(h, "POST", "/v1/trips", "owner", newTripBody); rec.Code != 201 {
		t.Fatalf("second trip: %d %s", rec.Code, rec.Body)
	}
	var tripID string
	{
		var tr api.Trip
		plans.upcoming = 0
		rec := call(h, "POST", "/v1/trips", "owner", newTripBody)
		_ = json.Unmarshal(rec.Body.Bytes(), &tr)
		tripID = tr.Id
	}
	plans.upcoming = 2
	rec := call(h, "POST", "/v1/trips", "owner", newTripBody)
	var e api.Error
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if rec.Code != 403 || e.Code != "limit_reached" || e.Feature == nil || *e.Feature != "upcomingTrips" {
		t.Fatalf("third trip: %d %s", rec.Code, rec.Body)
	}
	plans.plus = true
	if rec := call(h, "POST", "/v1/trips", "owner", newTripBody); rec.Code != 201 {
		t.Fatalf("Plus is not limited: %d", rec.Code)
	}
	plans.plus = false

	// Best order: one run per trip.
	path := "/v1/trips/" + tripID + "/days/2027-02-01/best-order"
	if rec := call(h, "POST", path, "owner", ""); rec.Code != 200 {
		t.Fatalf("first best order: %d %s", rec.Code, rec.Body)
	}
	rec = call(h, "POST", path, "owner", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if rec.Code != 403 || *e.Feature != "bestOrder" {
		t.Fatalf("second best order: %d %s", rec.Code, rec.Body)
	}

	// Imports: five bookings per trip, counted by what was found.
	imp := "/v1/trips/" + tripID + "/bookings/import"
	for i := range 5 {
		if rec := call(h, "POST", imp, "owner", `{"text":"x"}`); rec.Code != 200 {
			t.Fatalf("import %d: %d %s", i, rec.Code, rec.Body)
		}
	}
	rec = call(h, "POST", imp, "owner", `{"text":"x"}`)
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if rec.Code != 403 || *e.Feature != "forwardedBookings" {
		t.Fatalf("sixth import: %d %s", rec.Code, rec.Body)
	}
}
