package server

import (
	"context"
	"errors"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// PlanSource tells the entitlements routes who has Plus and what is used.
type PlanSource interface {
	// User returns the person's plan and their own counters.
	User(ctx context.Context, uid string) (entitlements.User, error)
	// Trip returns a trip's pass, crew Plus and counters.
	Trip(ctx context.Context, tripID string) (entitlements.Trip, error)
}

// WithEntitlements turns on the entitlements routes. Without it they answer 503.
func WithEntitlements(svc *entitlements.Service, src PlanSource) Option {
	return func(h *handlers) { h.plans, h.planSource = svc, src }
}

const msgNoPlans = "Plans are not available right now."

// GetMyEntitlements answers every feature for the caller with no trip in view.
func (h *handlers) GetMyEntitlements(ctx context.Context, _ api.GetMyEntitlementsRequestObject) (api.GetMyEntitlementsResponseObject, error) {
	id, err := identity(ctx, "GetMyEntitlements")
	if err != nil {
		return nil, err
	}
	if h.plans == nil {
		return api.GetMyEntitlements503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoPlans))}, nil
	}
	user, err := h.planSource.User(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	return api.GetMyEntitlements200JSONResponse(h.entitlementsFor(user, nil)), nil
}

// GetTripEntitlements answers every feature for the caller on one trip.
func (h *handlers) GetTripEntitlements(ctx context.Context, req api.GetTripEntitlementsRequestObject) (api.GetTripEntitlementsResponseObject, error) {
	id, err := identity(ctx, "GetTripEntitlements")
	if err != nil {
		return nil, err
	}
	if h.plans == nil || h.trips == nil {
		return api.GetTripEntitlements503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoPlans))}, nil
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.GetTripEntitlements404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	user, err := h.planSource.User(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	trip, err := h.planSource.Trip(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	return api.GetTripEntitlements200JSONResponse(h.entitlementsFor(user, &trip)), nil
}

func (h *handlers) entitlementsFor(user entitlements.User, trip *entitlements.Trip) api.Entitlements {
	out := api.Entitlements{Plan: api.Free, Features: map[string]api.EntitlementDecision{}}
	switch {
	case user.Plus || trip != nil && trip.MemberPlus:
		out.Plan = api.Plus
	case trip != nil && trip.Pass:
		out.Plan = api.Pass
	}
	for _, f := range entitlements.All() {
		d := h.plans.Check(user, trip, f)
		out.Features[string(f)] = api.EntitlementDecision{Allowed: d.Allowed, Limit: d.Limit, Used: d.Used, NeedsPlus: d.NeedsPlus}
	}
	return out
}
