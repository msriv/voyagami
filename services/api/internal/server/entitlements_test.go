package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
)

type fakePlans struct {
	user entitlements.User
	trip entitlements.Trip
}

func (f fakePlans) User(context.Context, string) (entitlements.User, error) { return f.user, nil }
func (f fakePlans) Trip(context.Context, string) (entitlements.Trip, error) { return f.trip, nil }

func TestEntitlementsRoutes(t *testing.T) {
	store := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner", "other": "u-other"}
	plans := fakePlans{trip: entitlements.Trip{Pass: true, PassBoughtBy: "u-other", PassEndsOn: "2026-11-12"}}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(store),
		WithEntitlements(entitlements.New(entitlements.Free()), plans))

	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)

	var me api.Entitlements
	rec := call(h, "GET", "/v1/me/entitlements", "owner", "")
	if rec.Code != 200 {
		t.Fatalf("me: %d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &me)
	if me.Plan != api.Free || !me.Features[string(entitlements.SettleUp)].Allowed || me.Features[string(entitlements.DarkMode)].Allowed {
		t.Fatalf("free account: %+v", me)
	}

	// Free limits ignore the caller's own plan.
	var free api.Entitlements
	plusH := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(store),
		WithEntitlements(entitlements.New(entitlements.Free()), fakePlans{user: entitlements.User{Plus: true}}))
	rec = call(plusH, "GET", "/v1/plans/free", "owner", "")
	if rec.Code != 200 {
		t.Fatalf("free plan: %d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &free)
	if free.Plan != api.Free || free.Features[string(entitlements.UpcomingTrips)].Limit != entitlements.Free()[entitlements.UpcomingTrips] {
		t.Fatalf("free plan: %+v", free)
	}
	if call(plusH, "GET", "/v1/plans/free", "", "").Code != 401 {
		t.Fatal("free plan limits need sign-in")
	}

	var tr api.Entitlements
	rec = call(h, "GET", "/v1/trips/"+trip.Id+"/entitlements", "owner", "")
	if rec.Code != 200 {
		t.Fatalf("trip: %d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &tr)
	if tr.Plan != api.Pass || !tr.Features[string(entitlements.MultiCurrency)].Allowed {
		t.Fatalf("pass trip: %+v", tr)
	}
	if tr.Pass == nil || tr.Pass.BoughtBy != "u-other" || tr.Pass.EndsOn == nil || *tr.Pass.EndsOn != "2026-11-12" {
		t.Fatalf("pass info: %+v", tr.Pass)
	}
	if me.Pass != nil {
		t.Fatal("the person's own answer has no pass")
	}
	if me.Subscription != nil {
		t.Fatal("a Free account has no subscription")
	}

	// Plus on its own: renewal date and whether it renews.
	ends := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	subH := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(store),
		WithEntitlements(entitlements.New(entitlements.Free()), fakePlans{user: entitlements.User{Plus: true,
			Sub: &entitlements.Subscription{EndsAt: ends, WillRenew: true, ProductID: "plus_yearly"}}}))
	var sub api.Entitlements
	_ = json.Unmarshal(call(subH, "GET", "/v1/me/entitlements", "owner", "").Body.Bytes(), &sub)
	if s := sub.Subscription; s == nil || s.State != api.Subscribed || !s.WillRenew || s.EndsAt == nil || !s.EndsAt.Equal(ends) || s.ProductId == nil || *s.ProductId != "plus_yearly" {
		t.Fatalf("subscription: %+v", sub.Subscription)
	}

	for _, c := range []struct {
		name, path, token string
		want              int
	}{
		{"signed out", "/v1/me/entitlements", "", 401},
		{"non-member", "/v1/trips/" + trip.Id + "/entitlements", "other", 404},
	} {
		if got := call(h, "GET", c.path, c.token, "").Code; got != c.want {
			t.Fatalf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestEntitlementsOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()))
	if got := call(h, "GET", "/v1/me/entitlements", "owner", "").Code; got != 503 {
		t.Fatalf("got %d, want 503", got)
	}
}
