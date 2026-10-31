package server

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/places"
)

// WithPlaces turns on place search. Without it the route answers 503.
func WithPlaces(s places.Searcher) Option { return func(h *handlers) { h.places = s } }

// SearchPlaces proxies a place search so the provider key stays server-side.
func (h *handlers) SearchPlaces(ctx context.Context, req api.SearchPlacesRequestObject) (api.SearchPlacesResponseObject, error) {
	if _, err := identity(ctx, "SearchPlaces"); err != nil {
		return nil, err
	}
	if h.places == nil {
		return api.SearchPlaces503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, "Place search is not available right now."))}, nil
	}
	q := strings.TrimSpace(req.Params.Q)
	if n := utf8.RuneCountInString(q); n < 2 || n > 100 {
		return api.SearchPlaces400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "q must be 2 to 100 characters.")}, nil
	}
	var near *places.Near
	if req.Params.Lat != nil && req.Params.Lng != nil {
		near = &places.Near{Lat: *req.Params.Lat, Lng: *req.Params.Lng}
	}
	found, err := h.places.Search(ctx, q, near)
	if err != nil {
		return api.SearchPlaces503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, "Place search is not available right now."))}, nil
	}
	out := api.PlaceSearchResults{Results: []api.PlaceSearchResult{}}
	for _, r := range found {
		if len(out.Results) == 10 {
			break
		}
		item := api.PlaceSearchResult{Name: r.Name, Lat: r.Lat, Lng: r.Lng}
		if r.Address != "" {
			a := r.Address
			item.Address = &a
		}
		if r.Category != "" {
			c := api.PlaceSearchResultCategory(r.Category)
			item.Category = &c
		}
		out.Results = append(out.Results, item)
	}
	return api.SearchPlaces200JSONResponse(out), nil
}

// ResolvePlaceLink reads a pin out of a pasted maps link.
func (h *handlers) ResolvePlaceLink(ctx context.Context, req api.ResolvePlaceLinkRequestObject) (api.ResolvePlaceLinkResponseObject, error) {
	if _, err := identity(ctx, "ResolvePlaceLink"); err != nil {
		return nil, err
	}
	if req.Body == nil || len(req.Body.Url) > 2000 {
		return api.ResolvePlaceLink400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "url is required and at most 2000 characters.")}, nil
	}
	r, err := places.Resolve(ctx, req.Body.Url)
	if err != nil {
		return api.ResolvePlaceLink404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "Couldn't find a place in that link. Try a Google Maps or OpenStreetMap link."))}, nil
	}
	return api.ResolvePlaceLink200JSONResponse(api.PlaceSearchResult{Name: r.Name, Lat: r.Lat, Lng: r.Lng}), nil
}
