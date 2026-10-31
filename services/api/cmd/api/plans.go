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
// the per-trip counters and Trip Passes (written by the RevenueCat webhook).
type planSource struct {
	accounts *firestorerepo.Entitlements
	trips    *firestorerepo.Trips
	billing  *firestorerepo.Billing
	// devPlan is "free" or "plus" to force every account's plan (dev only), else "".
	// The counters are still real, so a forced Free plan hits its limits.
	devPlan string
	// dev is true when APP_ENV is dev. Accounts without a mock plan then skip every
	// limit, and the mock accounts (see devplans.go) get the plan they stand for.
	dev bool
}

func (p planSource) User(ctx context.Context, uid string) (entitlements.User, error) {
	acct, err := p.accounts.AccountPlan(ctx, uid)
	if err != nil {
		return entitlements.User{}, err
	}
	plus, drafts := acct.Plus, acct.AskDrafts
	upcoming, err := p.upcomingOwned(ctx, uid)
	if err != nil {
		return entitlements.User{}, err
	}
	unlimited := false
	switch {
	case p.devPlan != "":
		plus = p.devPlan == "plus"
	case p.dev:
		dp, err := p.accounts.DevPlan(ctx, uid)
		if err != nil {
			return entitlements.User{}, err
		}
		unlimited = dp == ""
	}
	return entitlements.User{Plus: plus, Sub: acct.Sub, Unlimited: unlimited, Used: map[entitlements.Feature]int{
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
		if p.dev {
			dp, err := p.accounts.DevPlan(ctx, m.UID)
			if err != nil {
				return entitlements.Trip{}, err
			}
			if dp == devPlanPass {
				t.Pass, t.PassBoughtBy = true, m.UID
			}
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
	by, has, err := p.billing.ActivePass(ctx, tripID)
	if err != nil {
		return entitlements.Trip{}, err
	}
	if has && passCovers(trip.Dates.End, time.Now()) && !t.Pass {
		t.Pass, t.PassBoughtBy = true, by
	}
	if t.Pass {
		t.PassEndsOn = passEndsOn(trip.Dates.End)
	}
	return t, nil
}

// passGrace is how long a Trip Pass keeps working after the trip's last day,
// for jet lag, late expenses and the journal.
const passGrace = 2 * 24 * time.Hour

// passCovers reports whether a Trip Pass bought for a trip with this end date
// (YYYY-MM-DD) is still running at now. A trip with no end date, or one that
// cannot be read, keeps its pass. A pass belongs to one trip, so overlapping
// trips never share it.
func passCovers(end string, now time.Time) bool {
	day, err := time.Parse("2006-01-02", end)
	if end == "" || err != nil {
		return true
	}
	return now.Before(day.Add(24*time.Hour + passGrace))
}

// passEndsOn is the last day a pass works for a trip with this end date, or ""
// when the trip has no usable end date and so the pass has no end yet.
func passEndsOn(end string) string {
	day, err := time.Parse("2006-01-02", end)
	if end == "" || err != nil {
		return ""
	}
	return day.Add(passGrace).Format("2006-01-02")
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

// limitsFor is the Free table in every environment. Dev opens it per account
// (see planSource.User), not for everyone.
func limitsFor(config.Config) entitlements.Limits { return entitlements.Free() }
