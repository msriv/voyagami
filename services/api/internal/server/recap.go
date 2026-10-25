package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/recap"
)

// GetRecap assembles the recap for a trip. Stores that are off just leave
// their part out.
func (h *handlers) GetRecap(ctx context.Context, req api.GetRecapRequestObject) (api.GetRecapResponseObject, error) {
	if h.trips == nil {
		return api.GetRecap503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTrips))}, nil
	}
	uid, _, deny, err := h.crewRole(ctx, "GetRecap", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.GetRecap404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	trip, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	in := recap.Input{Trip: trip}
	if h.stops != nil {
		if in.Stops, err = h.stops.List(ctx, req.TripId); err != nil {
			return nil, err
		}
	}
	if h.journal != nil {
		all, err := h.journal.List(ctx, req.TripId)
		if err != nil {
			return nil, err
		}
		for _, e := range all {
			if e.Visibility == model.VisibilityPrivate && e.AuthorUID != uid {
				continue
			}
			in.Entries = append(in.Entries, e)
		}
	}
	if h.expenses != nil {
		list, err := h.expenses.List(ctx, req.TripId)
		if err != nil {
			return nil, err
		}
		for _, e := range list {
			in.SpentMinor += e.Settlement.AmountMinor
		}
		if !h.settleOff() {
			if in.Balances, err = h.balances(ctx, req.TripId); err != nil {
				return nil, err
			}
		}
	}

	r := recap.Build(in)
	out := api.Recap{
		TripId: req.TripId, Days: r.Days, Stops: r.Stops, Crew: r.Crew, DistanceMeters: r.DistanceMeters,
		SpentMinor: r.SpentMinor, Currency: r.Currency, TargetMinor: r.TargetMinor, RemainingMinor: r.RemainingMinor,
		SettledUp: r.SettledUp, Highlights: []api.RecapHighlight{},
	}
	if r.FavouriteDay != nil {
		out.FavouriteDay = &api.RecapDay{Date: r.FavouriteDay.Date, Entries: r.FavouriteDay.Count}
	}
	for _, hl := range r.Highlights {
		out.Highlights = append(out.Highlights, api.RecapHighlight{Date: hl.Date, AuthorUid: hl.AuthorUID, Text: hl.Text})
	}
	return api.GetRecap200JSONResponse(out), nil
}
