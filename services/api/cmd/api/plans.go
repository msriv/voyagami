package main

import (
	"context"
	"time"

	"github.com/msriv/voyagami/services/api/internal/config"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo/firestorerepo"
)

// planSource reads plan facts from Firestore: Plus, the per-person trip count
// and the per-trip counters. Trip Passes have no storage yet (D4 trip pass
// ticket), so they read as none.
type planSource struct {
	accounts *firestorerepo.Entitlements
	trips    *firestorerepo.Trips
	// devPlan is "free" or "plus" to force every account's plan (dev only), else "".
	// The counters are still real, so a forced Free plan hits its limits.
	devPlan string
}

func (p planSource) User(ctx context.Context, uid string) (entitlements.User, error) {
	plus, drafts, err := p.accounts.Account(ctx, uid)
	if err != nil {
		return entitlements.User{}, err
	}
	upcoming, err := p.upcomingOwned(ctx, uid)
	if err != nil {
		return entitlements.User{}, err
	}
	if p.devPlan != "" {
		plus = p.devPlan == "plus"
	}
	return entitlements.User{Plus: plus, Used: map[entitlements.Feature]int{
		entitlements.AskDrafts: drafts, entitlements.UpcomingTrips: upcoming,
	}}, nil
}

func (p planSource) Trip(ctx context.Context, tripID string) (entitlements.Trip, error) {
	trip, err := p.trips.Get(ctx, tripID)
	if err != nil {
		return entitlements.Trip{}, err
	}
	t := entitlements.Trip{Used: map[entitlements.Feature]int{
		entitlements.BestOrder:         trip.Usage.BestOrderRuns,
		entitlements.ForwardedBookings: trip.Usage.ForwardedBookings,
		entitlements.Files:             trip.Usage.Files,
		entitlements.JournalPhotos:     trip.Usage.Photos,
	}}
	if p.devPlan != "" {
		t.MemberPlus = p.devPlan == "plus"
		return t, nil
	}
	members, err := p.trips.Members(ctx, tripID)
	if err != nil {
		return entitlements.Trip{}, err
	}
	for _, m := range members {
		if m.Status != model.MemberActive {
			continue
		}
		plus, _, err := p.accounts.Account(ctx, m.UID)
		if err != nil {
			return entitlements.Trip{}, err
		}
		if plus {
			t.MemberPlus = true
			break
		}
	}
	return t, nil
}

// upcomingOwned counts the trips the person owns that are still to come. Trips
// they only joined are free, and past trips do not count.
func (p planSource) upcomingOwned(ctx context.Context, uid string) (int, error) {
	trips, err := p.trips.ListForMember(ctx, uid)
	if err != nil {
		return 0, err
	}
	today := time.Now().UTC().Format("2006-01-02")
	n := 0
	for _, t := range trips {
		if t.OwnerUID == uid && t.Status == model.TripPlanned && (t.Dates.End == "" || t.Dates.End >= today) {
			n++
		}
	}
	return n, nil
}

func limitsFor(cfg config.Config) entitlements.Limits {
	if cfg.DevPlan != "" {
		return entitlements.Free()
	}
	return entitlements.ForEnv(cfg.Env)
}
