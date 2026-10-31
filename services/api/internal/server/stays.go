package server

import (
	"context"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/repo"
	"github.com/msriv/voyagami/services/api/internal/stays"
)

// StayClicks counts taps on stay links per partner and day. It keeps nothing
// about the person, the trip or the place.
type StayClicks interface {
	Add(ctx context.Context, partnerID string, day time.Time) error
}

// WithStays turns on the stay link routes with the configured partners. Without
// it they answer 503. An empty partner list is fine and gives an empty list.
func WithStays(partners []stays.Partner, clicks StayClicks) Option {
	return func(h *handlers) { h.stayPartners, h.stayClicks, h.staysOn = partners, clicks, true }
}

const msgNoStays = "Stays are not available right now."

// ListStayLinks builds one hand-off link per partner for the trip.
func (h *handlers) ListStayLinks(ctx context.Context, req api.ListStayLinksRequestObject) (api.ListStayLinksResponseObject, error) {
	id, err := identity(ctx, "ListStayLinks")
	if err != nil {
		return nil, err
	}
	if !h.staysOn || h.trips == nil {
		return api.ListStayLinks503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoStays))}, nil
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.ListStayLinks404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	trip, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	if len(trip.Destinations) == 0 {
		return api.ListStayLinks400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "Add a destination to the trip to see stays.")}, nil
	}
	search := stays.Search{Destination: trip.Destinations[0].Name, CheckIn: trip.Dates.Start, CheckOut: trip.Dates.End}
	if req.Params.Guests != nil {
		search.Guests = *req.Params.Guests
	}
	links := make([]api.StayLink, 0, len(h.stayPartners))
	for _, p := range h.stayPartners {
		u, err := stays.Build(p, search)
		if err != nil {
			return nil, err
		}
		links = append(links, api.StayLink{PartnerId: p.ID, Name: p.Name, Url: u})
	}
	return api.ListStayLinks200JSONResponse(api.StayLinkList{Links: links}), nil
}

// RecordStayClick counts one tap on a partner's link.
func (h *handlers) RecordStayClick(ctx context.Context, req api.RecordStayClickRequestObject) (api.RecordStayClickResponseObject, error) {
	id, err := identity(ctx, "RecordStayClick")
	if err != nil {
		return nil, err
	}
	if !h.staysOn || h.trips == nil || h.stayClicks == nil {
		return api.RecordStayClick503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoStays))}, nil
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.RecordStayClick404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	known := false
	for _, p := range h.stayPartners {
		known = known || p.ID == req.Body.PartnerId
	}
	if !known {
		return api.RecordStayClick400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "Unknown partner.")}, nil
	}
	if err := h.stayClicks.Add(ctx, req.Body.PartnerId, time.Now()); err != nil {
		return nil, err
	}
	return api.RecordStayClick204Response{}, nil
}
