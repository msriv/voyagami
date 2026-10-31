package server

import (
	"context"
	"errors"
	"math"
	"sort"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/ranking"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const (
	defaultPickRadius = 10000
	defaultPickLimit  = 20
	maxPickLimit      = 50
	// candidatePool is how many nearby places are ranked. The picks come from the best of them.
	candidatePool = 200
)

// ListRecommendations ranks catalogue places for the trip's crew in one category.
func (h *handlers) ListRecommendations(ctx context.Context, req api.ListRecommendationsRequestObject) (api.ListRecommendationsResponseObject, error) {
	id, err := identity(ctx, "ListRecommendations")
	if err != nil {
		return nil, err
	}
	if h.catalog == nil || h.trips == nil || h.stops == nil {
		return api.ListRecommendations503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoCatalog))}, nil
	}
	if _, err := h.myMembership(ctx, req.TripId, id.UserID); errors.Is(err, repo.ErrNotFound) {
		return api.ListRecommendations404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	bad := func(msg string) (api.ListRecommendationsResponseObject, error) {
		return api.ListRecommendations400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	p := req.Params
	group := model.PlaceGroup(p.Category)
	if !model.ValidPlaceGroup(group) {
		return bad("Unknown category.")
	}
	radius, limit := float64(defaultPickRadius), defaultPickLimit
	if p.RadiusMeters != nil {
		if *p.RadiusMeters < 50 || *p.RadiusMeters > maxNearbyRadius {
			return bad("radiusMeters must be 50 to 20000.")
		}
		radius = float64(*p.RadiusMeters)
	}
	if p.Limit != nil {
		if *p.Limit < 1 || *p.Limit > maxPickLimit {
			return bad("limit must be 1 to 50.")
		}
		limit = *p.Limit
	}
	if (p.Lat == nil) != (p.Lng == nil) {
		return bad("Give both lat and lng, or neither.")
	}

	var anchorLat, anchorLng float64
	if p.Lat != nil {
		anchorLat, anchorLng = *p.Lat, *p.Lng
		if math.IsNaN(anchorLat) || math.IsNaN(anchorLng) || anchorLat < -90 || anchorLat > 90 || anchorLng < -180 || anchorLng > 180 {
			return bad("lat and lng must be a real position.")
		}
	} else {
		var ok bool
		if anchorLat, anchorLng, ok, err = h.tripCentre(ctx, req.TripId); err != nil {
			return nil, err
		} else if !ok {
			return bad("Add a place with a position to the trip, or give lat and lng.")
		}
	}

	found, _, err := h.catalog.Nearby(ctx, model.NearbyQuery{Lat: anchorLat, Lng: anchorLng, Radius: radius, Group: group, Limit: candidatePool})
	if err != nil {
		return nil, err
	}
	style, err := h.crewStyle(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]model.CatalogPlace, len(found))
	cands := make([]ranking.Place, 0, len(found))
	for _, n := range found {
		byID[n.ID] = n.CatalogPlace
		cands = append(cands, rankingPlace(n.CatalogPlace))
	}
	ranked := ranking.Rank(cands, ranking.Context{Style: style, AnchorLat: anchorLat, AnchorLng: anchorLng, HasAnchor: true, MinuteOfDay: -1})

	if maxPicks, err := h.pickCap(ctx, id.UserID, req.TripId); err != nil {
		return nil, err
	} else if maxPicks != entitlements.Unlimited && maxPicks < limit {
		limit = maxPicks
	}
	out := api.RecommendationList{Category: p.Category, Items: []api.Recommendation{}, HasMore: len(ranked) > limit}
	for _, r := range ranked {
		if len(out.Items) == limit {
			break
		}
		d := r.DistanceMeters
		item := api.Recommendation{Place: toAPIPlace(byID[r.Place.ID], trimmed(p.Lang), &d), Score: r.Score, Reasons: []api.RecommendationReason{}}
		for _, s := range r.Reasons {
			item.Reasons = append(item.Reasons, api.RecommendationReason(s))
		}
		out.Items = append(out.Items, item)
	}
	return api.ListRecommendations200JSONResponse(out), nil
}

// pickCap is how many picks the caller's plan shows per category, or Unlimited
// when limits are off.
func (h *handlers) pickCap(ctx context.Context, uid, tripID string) (int, error) {
	if h.plans == nil || h.planSource == nil {
		return entitlements.Unlimited, nil
	}
	user, err := h.planSource.User(ctx, uid)
	if err != nil {
		return 0, err
	}
	trip, err := h.planSource.Trip(ctx, tripID)
	if err != nil {
		return 0, err
	}
	return h.plans.PickLimit(user, &trip), nil
}

// tripCentre is the middle of the trip's stops that have a position. ok is
// false when none has one.
func (h *handlers) tripCentre(ctx context.Context, tripID string) (lat, lng float64, ok bool, err error) {
	stops, err := h.stops.List(ctx, tripID)
	if err != nil {
		return 0, 0, false, err
	}
	var n int
	for _, s := range stops {
		la, ln := s.Place.Lat, s.Place.Lng
		if !s.Place.Custom() {
			cp, err := h.catalog.Get(ctx, s.Place.PlaceID)
			if errors.Is(err, repo.ErrNotFound) {
				continue
			}
			if err != nil {
				return 0, 0, false, err
			}
			la, ln = cp.Lat, cp.Lng
		}
		if la == 0 && ln == 0 { // no position
			continue
		}
		lat, lng, n = lat+la, lng+ln, n+1
	}
	if n == 0 {
		return 0, 0, false, nil
	}
	return lat / float64(n), lng / float64(n), true, nil
}

// spendLevel maps the onboarding spend answer to the ranking scale.
var spendLevel = map[string]int{"shoestring": 1, "comfort": 2, "splurge": 3}

// crewStyle merges the active members' travel styles: the union of interests
// and the average spend.
func (h *handlers) crewStyle(ctx context.Context, tripID string) (ranking.Style, error) {
	if h.profile == nil {
		return ranking.Style{}, nil
	}
	members, err := h.trips.Members(ctx, tripID)
	if err != nil {
		return ranking.Style{}, err
	}
	interests := map[string]bool{}
	spendSum, spendN := 0, 0
	for _, m := range members {
		if m.Status != model.MemberActive {
			continue
		}
		u, err := h.profile.Get(ctx, m.UID)
		if errors.Is(err, repo.ErrNotFound) {
			continue
		}
		if err != nil {
			return ranking.Style{}, err
		}
		for _, i := range u.TravelStyle.Interests {
			interests[i] = true
		}
		if v, ok := spendLevel[u.TravelStyle.Spend]; ok {
			spendSum, spendN = spendSum+v, spendN+1
		}
	}
	st := ranking.Style{}
	for i := range interests {
		st.Interests = append(st.Interests, i)
	}
	sort.Strings(st.Interests)
	if spendN > 0 {
		st.Spend = int(math.Round(float64(spendSum) / float64(spendN)))
	}
	return st, nil
}

// rankingPlace turns a catalogue place into ranking input. Crew saves, opening
// hours and must-see flags are not stored yet, so they read as unknown.
func rankingPlace(p model.CatalogPlace) ranking.Place {
	kind := ""
	if len(p.Categories) > 0 {
		kind = p.Categories[0]
	}
	have := 0.0
	for _, ok := range []bool{len(p.Name) > 0, p.Address != "", len(p.Photos) > 0, len(p.Categories) > 0, p.TimeZone != ""} {
		if ok {
			have++
		}
	}
	return ranking.Place{
		ID: p.ID, Name: p.DisplayName("en"), Kind: kind, Tags: p.Tags, Lat: p.Lat, Lng: p.Lng, HasCoord: true,
		Rating: p.Rating, Reviews: p.Reviews, PriceLevel: p.PriceLevel, Completeness: have / 5,
	}
}
