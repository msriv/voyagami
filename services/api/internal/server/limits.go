package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithUsage turns on the per-trip usage counters that plan limits read.
func WithUsage(u repo.TripUsage) Option { return func(h *handlers) { h.usage = u } }

const codeLimitReached = "limit_reached"

// Plain-English reasons, for logs. The app shows its own copy.
const (
	msgTripLimit      = "The Free plan holds two upcoming trips. Plus lifts that."
	msgBestOrderLimit = "The Free plan includes one best-order run per trip. Plus lifts that."
	msgImportLimit    = "The Free plan includes five imported bookings per trip. Plus lifts that."
)

// limitBody is the 403 body for a plan limit. The app switches on the code and
// the feature to show the right paywall.
func limitBody(ctx context.Context, f entitlements.Feature, message string) api.Error {
	e := errBody(ctx, codeLimitReached, message)
	feature := string(f)
	e.Feature = &feature
	return e
}

// overLimit asks the entitlements service whether the person may use one more
// of a feature, on a trip or (tripID empty) for themselves. It returns the 403
// body when the answer is no, and nil when it is yes or limits are off.
func (h *handlers) overLimit(ctx context.Context, uid, tripID string, f entitlements.Feature, message string) (*api.Error, error) {
	if h.plans == nil || h.planSource == nil {
		return nil, nil
	}
	user, err := h.planSource.User(ctx, uid)
	if err != nil {
		return nil, err
	}
	var trip *entitlements.Trip
	if tripID != "" {
		t, err := h.planSource.Trip(ctx, tripID)
		if err != nil {
			return nil, err
		}
		trip = &t
	}
	if h.plans.Check(user, trip, f).Allowed {
		return nil, nil
	}
	e := limitBody(ctx, f, message)
	return &e, nil
}

// countUsage raises a trip counter. A failure is logged by the caller's
// error path; the action already happened, so it is not undone.
func (h *handlers) countUsage(ctx context.Context, tripID string, c model.UsageCounter, n int) error {
	if h.usage == nil || n <= 0 {
		return nil
	}
	return h.usage.Add(ctx, tripID, c, n)
}
