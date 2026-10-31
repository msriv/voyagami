package server

import (
	"context"
	"encoding/base64"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithCatalog turns on the places catalogue routes. Without it they answer 503.
func WithCatalog(c repo.PlaceCatalog) Option { return func(h *handlers) { h.catalog = c } }

const (
	msgNoCatalog        = "Places are not available right now."
	defaultNearbyRadius = 2000
	maxNearbyRadius     = 20000
	defaultNearbyLimit  = 20
	maxNearbyLimit      = 50
)

// GetCatalogPlace returns one catalogue place with its photos and sources.
func (h *handlers) GetCatalogPlace(ctx context.Context, req api.GetCatalogPlaceRequestObject) (api.GetCatalogPlaceResponseObject, error) {
	if _, err := identity(ctx, "GetCatalogPlace"); err != nil {
		return nil, err
	}
	if h.catalog == nil {
		return api.GetCatalogPlace503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCatalog))}, nil
	}
	p, err := h.catalog.Get(ctx, req.PlaceId)
	if errors.Is(err, repo.ErrNotFound) {
		return api.GetCatalogPlace404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such place."))}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetCatalogPlace200JSONResponse(toAPIPlace(p, trimmed(req.Params.Lang), nil)), nil
}

// ListNearbyPlaces returns catalogue places around a point, nearest first.
func (h *handlers) ListNearbyPlaces(ctx context.Context, req api.ListNearbyPlacesRequestObject) (api.ListNearbyPlacesResponseObject, error) {
	if _, err := identity(ctx, "ListNearbyPlaces"); err != nil {
		return nil, err
	}
	if h.catalog == nil {
		return api.ListNearbyPlaces503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCatalog))}, nil
	}
	bad := func(msg string) (api.ListNearbyPlacesResponseObject, error) {
		return api.ListNearbyPlaces400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	p := req.Params
	if math.IsNaN(p.Lat) || math.IsNaN(p.Lng) || p.Lat < -90 || p.Lat > 90 || p.Lng < -180 || p.Lng > 180 {
		return bad("lat and lng must be a real position.")
	}
	q := model.NearbyQuery{Lat: p.Lat, Lng: p.Lng, Radius: defaultNearbyRadius, Limit: defaultNearbyLimit}
	if p.RadiusMeters != nil {
		if *p.RadiusMeters < 50 || *p.RadiusMeters > maxNearbyRadius {
			return bad("radiusMeters must be 50 to 20000.")
		}
		q.Radius = float64(*p.RadiusMeters)
	}
	if p.Limit != nil {
		if *p.Limit < 1 || *p.Limit > maxNearbyLimit {
			return bad("limit must be 1 to 50.")
		}
		q.Limit = *p.Limit
	}
	if p.Group != nil {
		if !model.ValidPlaceGroup(model.PlaceGroup(*p.Group)) {
			return bad("Unknown group.")
		}
		q.Group = model.PlaceGroup(*p.Group)
	}
	if p.Cursor != nil {
		c, ok := decodeNearbyCursor(*p.Cursor)
		if !ok {
			return bad("The cursor is not valid.")
		}
		q.After = &c
	}
	found, more, err := h.catalog.Nearby(ctx, q)
	if err != nil {
		return nil, err
	}
	out := api.CatalogPlaceList{Places: make([]api.CatalogPlace, 0, len(found))}
	for _, n := range found {
		d := n.Distance
		out.Places = append(out.Places, toAPIPlace(n.CatalogPlace, trimmed(p.Lang), &d))
	}
	if more && len(found) > 0 {
		last := found[len(found)-1]
		c := encodeNearbyCursor(model.NearbyCursor{Distance: last.Distance, ID: last.ID})
		out.NextCursor = &c
	}
	return api.ListNearbyPlaces200JSONResponse(out), nil
}

func trimmed(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func toAPIPlace(p model.CatalogPlace, lang string, distance *float64) api.CatalogPlace {
	out := api.CatalogPlace{
		Id: p.ID, Name: p.DisplayName(lang), Lat: p.Lat, Lng: p.Lng, Group: api.PlaceGroup(p.Group),
		Categories: append([]string{}, p.Categories...), Photos: []api.CatalogPlacePhoto{}, Sources: []api.CatalogPlaceSource{},
	}
	if p.Address != "" {
		out.Address = &p.Address
	}
	if p.CountryCode != "" {
		out.CountryCode = &p.CountryCode
	}
	if p.TimeZone != "" {
		out.TimeZone = &p.TimeZone
	}
	for _, ph := range p.Photos {
		out.Photos = append(out.Photos, api.CatalogPlacePhoto{Url: ph.URL, Attribution: ph.Attribution, Source: ph.Source, Licence: ph.Licence})
	}
	if len(p.Hours) > 0 {
		hours := make([]api.OpeningPeriod, 0, len(p.Hours))
		for _, h := range p.Hours {
			hours = append(hours, api.OpeningPeriod{Day: api.OpeningPeriodDay(h.Day), Open: h.Open, Close: h.Close})
		}
		out.OpeningHours = &hours
	}
	fields := make([]string, 0, len(p.Provenance))
	for f := range p.Provenance {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, f := range fields {
		s := p.Provenance[f]
		item := api.CatalogPlaceSource{Field: f, Source: s.Source, Licence: s.Licence}
		if s.SourceID != "" {
			id := s.SourceID
			item.SourceId = &id
		}
		out.Sources = append(out.Sources, item)
	}
	if distance != nil {
		out.DistanceMeters = distance
	}
	return out
}

func encodeNearbyCursor(c model.NearbyCursor) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatFloat(c.Distance, 'g', -1, 64) + "|" + c.ID))
}

func decodeNearbyCursor(s string) (model.NearbyCursor, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return model.NearbyCursor{}, false
	}
	dist, id, ok := strings.Cut(string(raw), "|")
	d, err := strconv.ParseFloat(dist, 64)
	if !ok || err != nil || id == "" || math.IsNaN(d) || math.IsInf(d, 0) {
		return model.NearbyCursor{}, false
	}
	return model.NearbyCursor{Distance: d, ID: id}, true
}
