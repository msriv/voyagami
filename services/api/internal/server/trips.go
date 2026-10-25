package server

import (
	"context"
	"errors"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// TripStore is the trip storage the routes need. repo.Trips implements it.
type TripStore interface {
	Create(ctx context.Context, t model.Trip, owner model.Member) (model.Trip, error)
	Get(ctx context.Context, id string) (model.Trip, error)
	ListForMember(ctx context.Context, uid string) ([]model.Trip, error)
	Update(ctx context.Context, id string, patch model.TripPatch) (model.Trip, error)
	Member(ctx context.Context, tripID, uid string) (model.Member, error)
}

// WithTrips turns on the trip routes. Without it they answer 503.
func WithTrips(s TripStore) Option { return func(h *handlers) { h.trips = s } }

const msgRoleDenied = "Your role on this trip does not allow that."

const msgNoTrips = "Trips are not available right now."

func identity(ctx context.Context, op string) (auth.Identity, error) {
	id, ok := auth.FromContext(ctx)
	if !ok {
		// requireAuth guards these routes, so this means the wiring is broken.
		return id, errors.New(op + " reached without an identity")
	}
	return id, nil
}

func badRequest(err error) (string, bool) {
	var invalid *model.ValidationError
	if errors.As(err, &invalid) {
		return strings.Join(invalid.Problems, ". ") + ".", true
	}
	return "", false
}

// myMembership returns the caller's role on a trip, or repo.ErrNotFound when
// they are not an active member. Non-members and missing trips look the same.
func (h *handlers) myMembership(ctx context.Context, tripID, uid string) (model.Role, error) {
	m, err := h.trips.Member(ctx, tripID, uid)
	if err != nil {
		return "", err
	}
	if m.Status != model.MemberActive {
		return "", repo.ErrNotFound
	}
	return m.Role, nil
}

// ListTrips returns the caller's trips.
func (h *handlers) ListTrips(ctx context.Context, _ api.ListTripsRequestObject) (api.ListTripsResponseObject, error) {
	id, err := identity(ctx, "ListTrips")
	if err != nil {
		return nil, err
	}
	if h.trips == nil {
		return api.ListTrips503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTrips))}, nil
	}
	trips, err := h.trips.ListForMember(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	out := api.TripList{Trips: make([]api.Trip, 0, len(trips))}
	for _, t := range trips {
		role, err := h.myMembership(ctx, t.ID, id.UserID)
		if errors.Is(err, repo.ErrNotFound) {
			continue // left or removed: memberUids can lag a moment behind
		}
		if err != nil {
			return nil, err
		}
		out.Trips = append(out.Trips, tripFrom(t, role))
	}
	return api.ListTrips200JSONResponse(out), nil
}

// CreateTrip makes a trip with the caller as owner.
func (h *handlers) CreateTrip(ctx context.Context, req api.CreateTripRequestObject) (api.CreateTripResponseObject, error) {
	id, err := identity(ctx, "CreateTrip")
	if err != nil {
		return nil, err
	}
	if h.trips == nil {
		return api.CreateTrip503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTrips))}, nil
	}
	b := req.Body
	trip := model.Trip{
		ID:                 model.NewID(),
		Title:              strings.TrimSpace(b.Title),
		Destinations:       destinationsFrom(b.Destinations),
		Dates:              datesFrom(b.Dates),
		TimeZone:           b.TimeZone,
		SettlementCurrency: b.SettlementCurrency,
		Status:             model.TripPlanned,
	}
	created, err := h.trips.Create(ctx, trip, model.Member{UID: id.UserID, DisplayName: id.DisplayName})
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.CreateTrip400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, msg))}, nil
		}
		return nil, err
	}
	return api.CreateTrip201JSONResponse(tripFrom(created, model.RoleOwner)), nil
}

// GetTrip returns one trip to its members.
func (h *handlers) GetTrip(ctx context.Context, req api.GetTripRequestObject) (api.GetTripResponseObject, error) {
	id, err := identity(ctx, "GetTrip")
	if err != nil {
		return nil, err
	}
	if h.trips == nil {
		return api.GetTrip503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTrips))}, nil
	}
	role, err := h.myMembership(ctx, req.TripId, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return api.GetTrip404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	}
	if err != nil {
		return nil, err
	}
	t, err := h.trips.Get(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	return api.GetTrip200JSONResponse(tripFrom(t, role)), nil
}

// UpdateTrip changes a trip. Owners and editors change details; only the owner
// changes the status, which is how a trip is archived.
func (h *handlers) UpdateTrip(ctx context.Context, req api.UpdateTripRequestObject) (api.UpdateTripResponseObject, error) {
	id, err := identity(ctx, "UpdateTrip")
	if err != nil {
		return nil, err
	}
	if h.trips == nil {
		return api.UpdateTrip503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoTrips))}, nil
	}
	role, err := h.myMembership(ctx, req.TripId, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return api.UpdateTrip404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip."))}, nil
	}
	if err != nil {
		return nil, err
	}
	b := req.Body
	if role == model.RoleViewer || (b.Status != nil && role != model.RoleOwner) {
		return api.UpdateTrip403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, msgRoleDenied))}, nil
	}
	var patch model.TripPatch
	if b.Title != nil {
		title := strings.TrimSpace(*b.Title)
		patch.Title = &title
	}
	if b.Destinations != nil {
		ds := destinationsFrom(*b.Destinations)
		patch.Destinations = &ds
	}
	if b.Dates != nil {
		d := datesFrom(*b.Dates)
		patch.Dates = &d
	}
	patch.TimeZone = b.TimeZone
	if b.Status != nil {
		s := model.TripStatus(*b.Status)
		patch.Status = &s
	}
	updated, err := h.trips.Update(ctx, req.TripId, patch)
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.UpdateTrip400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, msg))}, nil
		}
		return nil, err
	}
	return api.UpdateTrip200JSONResponse(tripFrom(updated, role)), nil
}

func destinationsFrom(in []api.Destination) []model.Destination {
	out := make([]model.Destination, len(in))
	for i, d := range in {
		out[i] = model.Destination{Name: d.Name, CountryCode: d.CountryCode, TimeZone: d.TimeZone}
		if d.LocalCurrency != nil {
			out[i].LocalCurrency = *d.LocalCurrency
		}
	}
	return out
}

func datesFrom(in api.TripDates) model.TripDates {
	var d model.TripDates
	if in.Start != nil {
		d.Start = *in.Start
	}
	if in.End != nil {
		d.End = *in.End
	}
	if f := in.Flexible; f != nil {
		d.Flexible = &model.FlexibleDates{Days: f.Days}
		if f.Month != nil {
			d.Flexible.Month = *f.Month
		}
	}
	return d
}

func tripFrom(t model.Trip, role model.Role) api.Trip {
	out := api.Trip{
		Id:                 t.ID,
		Title:              t.Title,
		OwnerId:            t.OwnerUID,
		TimeZone:           t.TimeZone,
		SettlementCurrency: t.SettlementCurrency,
		Status:             api.TripStatus(t.Status),
		Role:               api.TripRole(role),
		Destinations:       make([]api.Destination, len(t.Destinations)),
	}
	for i, d := range t.Destinations {
		out.Destinations[i] = api.Destination{Name: d.Name, CountryCode: d.CountryCode, TimeZone: d.TimeZone}
		if d.LocalCurrency != "" {
			c := d.LocalCurrency
			out.Destinations[i].LocalCurrency = &c
		}
	}
	if t.Dates.Start != "" || t.Dates.End != "" {
		start, end := t.Dates.Start, t.Dates.End
		out.Dates.Start, out.Dates.End = &start, &end
	}
	if f := t.Dates.Flexible; f != nil {
		out.Dates.Flexible = &struct {
			Days  int     `json:"days"`
			Month *string `json:"month,omitempty"`
		}{Days: f.Days}
		if f.Month != "" {
			m := f.Month
			out.Dates.Flexible.Month = &m
		}
	}
	return out
}
