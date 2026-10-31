package server

import (
	"context"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
	"github.com/msriv/voyagami/services/api/internal/stays"
)

// WithExperiences turns on the experience routes with the configured partners.
// It reuses the stay click counter; partner ids are stored with an "exp-"
// prefix so the two never mix. It also needs the catalogue, stops and trips.
func WithExperiences(partners []stays.Partner, clicks StayClicks) Option {
	return func(h *handlers) { h.expPartners, h.expClicks, h.expOn = partners, clicks, true }
}

const (
	msgNoExperiences = "Experiences are not available right now."
	expRadius        = 10000
	expLimit         = 20
)

// ListExperiences lists nearby "do" places with a hand-off link per partner.
func (h *handlers) ListExperiences(ctx context.Context, req api.ListExperiencesRequestObject) (api.ListExperiencesResponseObject, error) {
	id, err := identity(ctx, "ListExperiences")
	if err != nil {
		return nil, err
	}
	if !h.expOn || h.catalog == nil || h.stops == nil || h.trips == nil {
		return api.ListExperiences503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExperiences))}, nil
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.ListExperiences404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	out := api.ExperienceList{Items: []api.Experience{}}
	lat, lng, ok, err := h.tripCentre(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return api.ListExperiences200JSONResponse(out), nil
	}
	found, _, err := h.catalog.Nearby(ctx, model.NearbyQuery{Lat: lat, Lng: lng, Radius: expRadius, Group: model.GroupDo, Limit: expLimit})
	if err != nil {
		return nil, err
	}
	for _, n := range found {
		name := n.DisplayName("en")
		item := api.Experience{PlaceId: n.ID, Name: name, DistanceMeters: int(n.Distance), Links: []api.ExperienceLink{}}
		if len(n.Photos) > 0 {
			ph := n.Photos[0]
			item.Photo = &api.CatalogPlacePhoto{Url: ph.URL, Attribution: ph.Attribution, Source: ph.Source, Licence: ph.Licence}
		}
		for _, p := range h.expPartners {
			u, err := stays.Build(p, stays.Search{Destination: name})
			if err != nil {
				return nil, err
			}
			item.Links = append(item.Links, api.ExperienceLink{PartnerId: p.ID, Name: p.Name, Url: u})
		}
		out.Items = append(out.Items, item)
	}
	return api.ListExperiences200JSONResponse(out), nil
}

// RecordExperienceClick counts one tap on a partner's link.
func (h *handlers) RecordExperienceClick(ctx context.Context, req api.RecordExperienceClickRequestObject) (api.RecordExperienceClickResponseObject, error) {
	id, err := identity(ctx, "RecordExperienceClick")
	if err != nil {
		return nil, err
	}
	if !h.expOn || h.trips == nil || h.expClicks == nil {
		return api.RecordExperienceClick503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExperiences))}, nil
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.RecordExperienceClick404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	known := false
	for _, p := range h.expPartners {
		known = known || p.ID == req.Body.PartnerId
	}
	if !known {
		return api.RecordExperienceClick400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "Unknown partner.")}, nil
	}
	if err := h.expClicks.Add(ctx, "exp-"+req.Body.PartnerId, time.Now()); err != nil {
		return nil, err
	}
	return api.RecordExperienceClick204Response{}, nil
}
