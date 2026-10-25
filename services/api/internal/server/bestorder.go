package server

import (
	"context"
	"errors"
	"math"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/bestorder"
	"github.com/msriv/voyagami/services/api/internal/fracindex"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// cityKmPerHour is the assumed speed for straight-line travel estimates until
// the directions service replaces them.
const cityKmPerHour = 20

// dayStops returns the trip's live stops on one day, in order.
func (h *handlers) dayStops(ctx context.Context, tripID, date string) ([]model.Stop, error) {
	all, err := h.stops.List(ctx, tripID)
	if err != nil {
		return nil, err
	}
	var day []model.Stop
	for _, s := range all {
		if s.Date == date {
			day = append(day, s)
		}
	}
	return day, nil
}

// SuggestBestOrder proposes a shorter order for a day. It changes nothing.
func (h *handlers) SuggestBestOrder(ctx context.Context, req api.SuggestBestOrderRequestObject) (api.SuggestBestOrderResponseObject, error) {
	if h.stopsOff() {
		return api.SuggestBestOrder503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoStops))}, nil
	}
	_, deny, err := h.stopAccess(ctx, req.TripId, false)
	if err != nil {
		return nil, err
	}
	if _, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.SuggestBestOrder404JSONResponse{NotFoundJSONResponse: errNotFound(ctx)}, nil
	}
	if !model.ValidDate(req.Date) {
		return api.SuggestBestOrder400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "date must be YYYY-MM-DD.")}, nil
	}
	day, err := h.dayStops(ctx, req.TripId, req.Date)
	if err != nil {
		return nil, err
	}

	// A stop with a start time keeps its place, and so does one with no map
	// position, because we cannot say how far it is from the others.
	points := map[string]bestorder.Point{}
	in := make([]bestorder.Stop, len(day))
	for i, s := range day {
		in[i] = bestorder.Stop{ID: s.ID, Pinned: s.StartLocal != "" || !s.Place.Custom()}
		if s.Place.Custom() {
			points[s.ID] = bestorder.Point{Lat: s.Place.Lat, Lng: s.Place.Lng}
		} else {
			points[s.ID] = bestorder.Point{} // pinned, so its position is never used to move it
		}
	}
	res, err := bestorder.Suggest(ctx, in, bestorder.Straight{Points: points, KmPerHour: cityKmPerHour})
	if err != nil {
		return nil, err
	}
	return api.SuggestBestOrder200JSONResponse(api.BestOrder{
		Date: req.Date, Changed: res.Changed, Approx: true, StopIds: res.Order,
		BeforeMin: int(math.Round(res.BeforeMin)), AfterMin: int(math.Round(res.AfterMin)),
	}), nil
}

// SetDayOrder puts a day's stops in the given order.
func (h *handlers) SetDayOrder(ctx context.Context, req api.SetDayOrderRequestObject) (api.SetDayOrderResponseObject, error) {
	if h.stopsOff() {
		return api.SetDayOrder503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoStops))}, nil
	}
	_, deny, err := h.stopAccess(ctx, req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.SetDayOrder404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.SetDayOrder403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	if !model.ValidDate(req.Date) {
		return api.SetDayOrder400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "date must be YYYY-MM-DD.")}, nil
	}
	day, err := h.dayStops(ctx, req.TripId, req.Date)
	if err != nil {
		return nil, err
	}
	want := req.Body.StopIds
	if !sameStops(day, want) {
		return api.SetDayOrder400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "stopIds must list every stop on that day exactly once.")}, nil
	}
	orders := make(map[string]string, len(want))
	last := ""
	for _, id := range want {
		last, err = fracindex.Between(last, "")
		if err != nil {
			return nil, err
		}
		orders[id] = last
	}
	if err := h.stops.SetOrders(ctx, req.TripId, orders); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return api.SetDayOrder404JSONResponse{NotFoundJSONResponse: errNotFound(ctx)}, nil
		}
		return nil, err
	}
	day, err = h.dayStops(ctx, req.TripId, req.Date)
	if err != nil {
		return nil, err
	}
	out := api.StopList{Stops: make([]api.Stop, len(day))}
	for i, s := range day {
		out.Stops[i] = stopFrom(s)
	}
	return api.SetDayOrder200JSONResponse(out), nil
}

// sameStops reports whether ids lists each stop of the day exactly once.
func sameStops(day []model.Stop, ids []string) bool {
	if len(day) != len(ids) {
		return false
	}
	have := make(map[string]bool, len(day))
	for _, s := range day {
		have[s.ID] = true
	}
	for _, id := range ids {
		if !have[id] {
			return false
		}
		delete(have, id)
	}
	return true
}
