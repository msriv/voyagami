package server

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithSponsored turns on sponsored cards. Without it every route answers 503.
func WithSponsored(s repo.Sponsored) Option { return func(h *handlers) { h.sponsored = s } }

const msgNoSponsored = "Sponsored cards are not available right now."

func (h *handlers) sponsoredOff() bool { return h.sponsored == nil }

// GetSponsoredCard returns the one card that fits the trip's destination and
// the list, or none. It is the same on every plan, Plus included.
func (h *handlers) GetSponsoredCard(ctx context.Context, req api.GetSponsoredCardRequestObject) (api.GetSponsoredCardResponseObject, error) {
	id, err := identity(ctx, "GetSponsoredCard")
	if err != nil {
		return nil, err
	}
	if h.sponsoredOff() || h.trips == nil {
		return api.GetSponsoredCard503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSponsored))}, nil
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.GetSponsoredCard404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	trip, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	var group model.PlaceGroup
	if req.Params.Group != nil {
		group = model.PlaceGroup(*req.Params.Group)
	}
	now := time.Now()
	live, err := h.sponsored.Live(ctx, now)
	if err != nil {
		return nil, err
	}
	hidden, err := h.sponsored.Hidden(ctx, id.UserID, now)
	if err != nil {
		return nil, err
	}
	var fits []model.SponsoredCard
	for _, c := range live {
		if !hidden[c.ID] && c.Runs(now, trip.DestinationCountries, group) {
			fits = append(fits, c)
		}
	}
	if len(fits) == 0 {
		return api.GetSponsoredCard200JSONResponse(api.SponsoredResult{}), nil
	}
	sort.Slice(fits, func(i, j int) bool {
		if fits[i].Priority != fits[j].Priority {
			return fits[i].Priority > fits[j].Priority
		}
		return fits[i].ID < fits[j].ID
	})
	return api.GetSponsoredCard200JSONResponse(api.SponsoredResult{Card: toAPISponsored(fits[0])}), nil
}

func toAPISponsored(c model.SponsoredCard) *api.SponsoredCard {
	out := api.SponsoredCard{Id: c.ID, Sponsor: c.Sponsor, Title: c.Title, Url: c.URL}
	if c.Body != "" {
		out.Body = &c.Body
	}
	if c.ImageURL != "" {
		out.ImageUrl = &c.ImageURL
	}
	return &out
}

// RecordSponsoredImpression counts a card that was shown.
func (h *handlers) RecordSponsoredImpression(ctx context.Context, req api.RecordSponsoredImpressionRequestObject) (api.RecordSponsoredImpressionResponseObject, error) {
	id, err := identity(ctx, "RecordSponsoredImpression")
	if err != nil {
		return nil, err
	}
	if h.sponsoredOff() {
		return api.RecordSponsoredImpression503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSponsored))}, nil
	}
	if ok, err := h.cardExists(ctx, req.CardId); err != nil {
		return nil, err
	} else if !ok {
		return api.RecordSponsoredImpression404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such card."))}, nil
	}
	if err := h.sponsored.Record(ctx, id.UserID, req.CardId, model.SponsoredImpression, time.Now()); err != nil {
		return nil, err
	}
	return api.RecordSponsoredImpression204Response{}, nil
}

// RecordSponsoredClick counts a card that was tapped.
func (h *handlers) RecordSponsoredClick(ctx context.Context, req api.RecordSponsoredClickRequestObject) (api.RecordSponsoredClickResponseObject, error) {
	id, err := identity(ctx, "RecordSponsoredClick")
	if err != nil {
		return nil, err
	}
	if h.sponsoredOff() {
		return api.RecordSponsoredClick503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSponsored))}, nil
	}
	if ok, err := h.cardExists(ctx, req.CardId); err != nil {
		return nil, err
	} else if !ok {
		return api.RecordSponsoredClick404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such card."))}, nil
	}
	if err := h.sponsored.Record(ctx, id.UserID, req.CardId, model.SponsoredClick, time.Now()); err != nil {
		return nil, err
	}
	return api.RecordSponsoredClick204Response{}, nil
}

// HideSponsoredCard keeps a card away from this person for 30 days.
func (h *handlers) HideSponsoredCard(ctx context.Context, req api.HideSponsoredCardRequestObject) (api.HideSponsoredCardResponseObject, error) {
	id, err := identity(ctx, "HideSponsoredCard")
	if err != nil {
		return nil, err
	}
	if h.sponsoredOff() {
		return api.HideSponsoredCard503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSponsored))}, nil
	}
	if ok, err := h.cardExists(ctx, req.CardId); err != nil {
		return nil, err
	} else if !ok {
		return api.HideSponsoredCard404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such card."))}, nil
	}
	until := time.Now().Add(model.HideSponsoredFor)
	if err := h.sponsored.Hide(ctx, id.UserID, req.CardId, until); err != nil {
		return nil, err
	}
	return api.HideSponsoredCard200JSONResponse(api.SponsoredHidden{HiddenUntil: until}), nil
}

func (h *handlers) cardExists(ctx context.Context, cardID string) (bool, error) {
	_, err := h.sponsored.Get(ctx, cardID)
	if errors.Is(err, repo.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}
