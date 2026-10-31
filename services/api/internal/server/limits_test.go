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

// A Plus person who used more than the Free limits downgrades. Nothing is
// deleted or hidden; the extra use is refused and the entitlements say read-only.
func TestDowngradeKeepsContentAndRefusesMore(t *testing.T) {
	plans := &livePlans{plus: true, usage: map[model.UsageCounter]int{}}
	trips := newFakeTrips()
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(trips),
		WithStops(&fakeStops{stops: map[string]model.Stop{}}),
		WithBookings(&fakeBookings{byID: map[string]model.Booking{}}), WithImporter(oneBooking{}),
		WithUsage(plans), WithEntitlements(entitlements.New(entitlements.Free()), plans))
	var tr api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &tr)
	imp := "/v1/trips/" + tr.Id + "/bookings/import"

	// A booking the person saved while on Plus.
	if rec := call(h, "POST", "/v1/trips/"+tr.Id+"/bookings", "owner", `{"kind":"stay","title":"Old Quarter hotel"}`); rec.Code != 201 {
		t.Fatalf("save booking: %d %s", rec.Code, rec.Body)
	}

	// On Plus: seven imports and three best-order runs, all above the Free limits.
	for range 7 {
		if rec := call(h, "POST", imp, "owner", `{"text":"x"}`); rec.Code != 200 {
			t.Fatalf("plus import: %d %s", rec.Code, rec.Body)
		}
	}
	day := "/v1/trips/" + tr.Id + "/days/2027-02-01/best-order"
	for range 3 {
		if rec := call(h, "POST", day, "owner", ""); rec.Code != 200 {
			t.Fatalf("plus best order: %d %s", rec.Code, rec.Body)
		}
	}
	list := func() int {
		var out struct {
			Bookings []json.RawMessage `json:"bookings"`
		}
		rec := call(h, "GET", "/v1/trips/"+tr.Id+"/bookings", "owner", "")
		if rec.Code != 200 {
			t.Fatalf("list bookings: %d %s", rec.Code, rec.Body)
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return len(out.Bookings)
	}
	before := list()

	// Downgrade.
	plans.plus = false
	if after := list(); after != before || before == 0 {
		t.Fatalf("bookings before %d, after the downgrade %d: nothing may be deleted or hidden", before, after)
	}
	for _, c := range []struct{ name, path, body, feature string }{
		{"import", imp, `{"text":"x"}`, "forwardedBookings"},
		{"best order", day, "", "bestOrder"},
	} {
		rec := call(h, "POST", c.path, "owner", c.body)
		var e api.Error
		_ = json.Unmarshal(rec.Body.Bytes(), &e)
		if rec.Code != 403 || e.Feature == nil || *e.Feature != c.feature {
			t.Errorf("%s after the downgrade: %d %s", c.name, rec.Code, rec.Body)
		}
	}
	var ent api.Entitlements
	rec := call(h, "GET", "/v1/trips/"+tr.Id+"/entitlements", "owner", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &ent)
	for _, f := range []entitlements.Feature{entitlements.ForwardedBookings, entitlements.BestOrder} {
		d := ent.Features[string(f)]
		if d.Allowed || !d.ReadOnly {
			t.Errorf("%s: %+v, want not allowed and read-only", f, d)
		}
	}
}
