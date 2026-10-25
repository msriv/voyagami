package main

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/config"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo/firestorerepo"
)

// planSource reads plan facts from Firestore. Trip Passes and the per-trip
// counters have no storage yet (they arrive with the limits epic), so they
// read as none and zero; dev has no limits anyway.
type planSource struct {
	accounts *firestorerepo.Entitlements
	trips    *firestorerepo.Trips
	// devPlan is "free" or "plus" to force every account (dev only), else "".
	devPlan string
}

func (p planSource) User(ctx context.Context, uid string) (entitlements.User, error) {
	if p.devPlan != "" {
		return entitlements.User{Plus: p.devPlan == "plus"}, nil
	}
	plus, drafts, err := p.accounts.Account(ctx, uid)
	if err != nil {
		return entitlements.User{}, err
	}
	return entitlements.User{Plus: plus, Used: map[entitlements.Feature]int{entitlements.AskDrafts: drafts}}, nil
}

func (p planSource) Trip(ctx context.Context, tripID string) (entitlements.Trip, error) {
	if p.devPlan != "" {
		return entitlements.Trip{MemberPlus: p.devPlan == "plus"}, nil
	}
	members, err := p.trips.Members(ctx, tripID)
	if err != nil {
		return entitlements.Trip{}, err
	}
	t := entitlements.Trip{}
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

// limitsFor picks the free limits to enforce: ForEnv, except a DEV_PLAN
// override turns the free caps on in dev so they can be tried.
func limitsFor(cfg config.Config) entitlements.Limits {
	if cfg.DevPlan != "" {
		return entitlements.Free()
	}
	return entitlements.ForEnv(cfg.Env)
}
