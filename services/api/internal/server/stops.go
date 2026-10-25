package server

import (
	"context"
	"errors"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/fracindex"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// StopStore is the stop storage the routes need. repo.Stops implements it.
type StopStore = repo.Stops

// WithStops turns on the stop routes. They also need WithTrips. Without both they answer 503.
func WithStops(s StopStore) Option { return func(h *handlers) { h.stops = s } }

const msgNoStops = "The itinerary is not available right now."

// stopAccess checks the caller can use the trip's stops. A non-nil response is the answer to send.
func (h *handlers) stopAccess(ctx context.Context, tripID string, write bool) (model.Trip, any, error) {
	id, err := identity(ctx, "stops")
	if err != nil {
		return model.Trip{}, nil, err
	}
	role, err := h.myMembership(ctx, tripID, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return model.Trip{}, errNotFound(ctx), nil
	}
	if err != nil {
		return model.Trip{}, nil, err
	}
	if write && role == model.RoleViewer {
		return model.Trip{}, errForbidden(ctx), nil
	}
	trip, err := h.trips.Get(ctx, tripID)
	return trip, nil, err
}

func errNotFound(ctx context.Context) api.NotFoundJSONResponse {
	return api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip or stop."))
}

func errForbidden(ctx context.Context) api.ForbiddenJSONResponse {
	return api.ForbiddenJSONResponse(errBody(ctx, codeForbidden, "Your role on this trip does not allow that."))
}

func errInvalid(ctx context.Context, msg string) api.BadRequestJSONResponse {
	return api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, msg))
}

func (h *handlers) stopsOff() bool { return h.trips == nil || h.stops == nil }

// ListStops returns a trip's stops in day and order.
func (h *handlers) ListStops(ctx context.Context, req api.ListStopsRequestObject) (api.ListStopsResponseObject, error) {
	if h.stopsOff() {
		return api.ListStops503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoStops))}, nil
	}
	_, deny, err := h.stopAccess(ctx, req.TripId, false)
	if err != nil {
		return nil, err
	}
	if deny != nil {
		return api.ListStops404JSONResponse{NotFoundJSONResponse: errNotFound(ctx)}, nil
	}
	stops, err := h.stops.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.StopList{Stops: make([]api.Stop, len(stops))}
	for i, s := range stops {
		out.Stops[i] = stopFrom(s)
	}
	return api.ListStops200JSONResponse(out), nil
}

// CreateStop adds a stop to a day.
func (h *handlers) CreateStop(ctx context.Context, req api.CreateStopRequestObject) (api.CreateStopResponseObject, error) {
	if h.stopsOff() {
		return api.CreateStop503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoStops))}, nil
	}
	trip, deny, err := h.stopAccess(ctx, req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.CreateStop404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.CreateStop403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	b := req.Body
	stop := model.Stop{
		ID: model.NewID(), TripID: trip.ID, Date: b.Date, TimeZone: trip.TimeZone,
		Place: placeFrom(b.Place), StartLocal: strOr(b.StartLocal), DurationMin: intOr0(b.DurationMin), Note: strOr(b.Note),
	}
	if msg := dateOutsideTrip(trip, b.Date); msg != "" {
		return api.CreateStop400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	if stop.Order, err = h.orderFor(ctx, trip.ID, b.Date, strOr(b.BeforeStopId), ""); err != nil {
		return h.createStopError(ctx, err)
	}
	created, err := h.stops.Create(ctx, stop)
	if err != nil {
		return h.createStopError(ctx, err)
	}
	return api.CreateStop201JSONResponse(stopFrom(created)), nil
}

func (h *handlers) createStopError(ctx context.Context, err error) (api.CreateStopResponseObject, error) {
	if msg, ok := stopProblem(err); ok {
		return api.CreateStop400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	return nil, err
}

// UpdateStop changes or moves a stop. Only the moved stop is written.
func (h *handlers) UpdateStop(ctx context.Context, req api.UpdateStopRequestObject) (api.UpdateStopResponseObject, error) {
	if h.stopsOff() {
		return api.UpdateStop503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoStops))}, nil
	}
	trip, deny, err := h.stopAccess(ctx, req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.UpdateStop404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.UpdateStop403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	cur, err := h.liveStop(ctx, req.TripId, req.StopId)
	if errors.Is(err, repo.ErrNotFound) {
		return api.UpdateStop404JSONResponse{NotFoundJSONResponse: errNotFound(ctx)}, nil
	}
	if err != nil {
		return nil, err
	}
	b := req.Body
	patch := model.StopPatch{StartLocal: b.StartLocal, DurationMin: b.DurationMin, Note: b.Note}
	if b.Date != nil {
		if msg := dateOutsideTrip(trip, *b.Date); msg != "" {
			return api.UpdateStop400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
	}
	day := cur.Date
	if b.Date != nil {
		day = *b.Date
		patch.Date = b.Date
	}
	// The order changes when the stop goes before another, or to a different day.
	if before := strOr(b.BeforeStopId); before != "" || day != cur.Date {
		order, err := h.orderFor(ctx, req.TripId, day, before, cur.ID)
		if err != nil {
			return h.updateStopError(ctx, err)
		}
		patch.Order = &order
	}
	updated, err := h.stops.Update(ctx, req.TripId, req.StopId, patch)
	if err != nil {
		return h.updateStopError(ctx, err)
	}
	return api.UpdateStop200JSONResponse(stopFrom(updated)), nil
}

func (h *handlers) updateStopError(ctx context.Context, err error) (api.UpdateStopResponseObject, error) {
	if msg, ok := stopProblem(err); ok {
		return api.UpdateStop400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	if errors.Is(err, repo.ErrNotFound) {
		return api.UpdateStop404JSONResponse{NotFoundJSONResponse: errNotFound(ctx)}, nil
	}
	return nil, err
}

// DeleteStop hides a stop. It is kept so it can be restored.
func (h *handlers) DeleteStop(ctx context.Context, req api.DeleteStopRequestObject) (api.DeleteStopResponseObject, error) {
	if h.stopsOff() {
		return api.DeleteStop503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoStops))}, nil
	}
	_, deny, err := h.stopAccess(ctx, req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.DeleteStop404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.DeleteStop403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	if _, err := h.liveStop(ctx, req.TripId, req.StopId); errors.Is(err, repo.ErrNotFound) {
		return api.DeleteStop404JSONResponse{NotFoundJSONResponse: errNotFound(ctx)}, nil
	} else if err != nil {
		return nil, err
	}
	if _, err := h.stops.Update(ctx, req.TripId, req.StopId, model.StopPatch{Deleted: true}); err != nil {
		return nil, err
	}
	return api.DeleteStop204Response{}, nil
}

// liveStop returns the stop, treating a deleted one as missing.
func (h *handlers) liveStop(ctx context.Context, tripID, stopID string) (model.Stop, error) {
	s, err := h.stops.Get(ctx, tripID, stopID)
	if err == nil && s.DeletedAt != nil {
		return model.Stop{}, repo.ErrNotFound
	}
	return s, err
}

// errBeforeStop means beforeStopId is not another stop on the target day.
var errBeforeStop = errors.New("beforeStopId must be another stop on that day")

// orderFor picks a sort key that puts a stop on the given day just before
// beforeID, or at the end of the day when that is empty. skipID is the stop
// being moved, so it does not count as its own neighbour.
func (h *handlers) orderFor(ctx context.Context, tripID, date, beforeID, skipID string) (string, error) {
	stops, err := h.stops.List(ctx, tripID)
	if err != nil {
		return "", err
	}
	var day []model.Stop
	for _, s := range stops {
		if s.Date == date && s.ID != skipID {
			day = append(day, s)
		}
	}
	lo, hi := "", ""
	if beforeID == "" {
		if n := len(day); n > 0 {
			lo = day[n-1].Order
		}
	} else {
		i := indexOfStop(day, beforeID)
		if i < 0 {
			return "", errBeforeStop
		}
		hi = day[i].Order
		if i > 0 {
			lo = day[i-1].Order
		}
	}
	return fracindex.Between(lo, hi)
}

func indexOfStop(stops []model.Stop, id string) int {
	for i, s := range stops {
		if s.ID == id {
			return i
		}
	}
	return -1
}

// stopProblem turns a validation or position error into a message for the person.
func stopProblem(err error) (string, bool) {
	if errors.Is(err, errBeforeStop) {
		return err.Error() + ".", true
	}
	return badRequest(err)
}

// dateOutsideTrip returns a problem when the trip has fixed dates and the day is not inside them.
func dateOutsideTrip(t model.Trip, date string) string {
	if t.Dates.Start != "" && t.Dates.End != "" && (date < t.Dates.Start || date > t.Dates.End) {
		return "date must be within the trip's dates."
	}
	return ""
}

func placeFrom(p api.PlaceRef) model.PlaceRef {
	return model.PlaceRef{
		PlaceID: strings.TrimSpace(strOr(p.PlaceId)), Name: strings.TrimSpace(strOr(p.Name)),
		Lat: floatOr0(p.Lat), Lng: floatOr0(p.Lng),
		Address: strings.TrimSpace(strOr(p.Address)), Category: string(categoryOr(p.Category)),
	}
}

func placeTo(p model.PlaceRef) api.PlaceRef {
	if !p.Custom() {
		id := p.PlaceID
		return api.PlaceRef{PlaceId: &id}
	}
	name, lat, lng := p.Name, p.Lat, p.Lng
	out := api.PlaceRef{Name: &name, Lat: &lat, Lng: &lng}
	if a := p.Address; a != "" {
		out.Address = &a
	}
	if c := api.PlaceRefCategory(p.Category); c != "" {
		out.Category = &c
	}
	return out
}

func stopFrom(s model.Stop) api.Stop {
	out := api.Stop{Id: s.ID, TripId: s.TripID, Date: s.Date, Order: s.Order, TimeZone: s.TimeZone}
	out.Place = placeTo(s.Place)
	if s.StartLocal != "" {
		v := s.StartLocal
		out.StartLocal = &v
	}
	if s.DurationMin > 0 {
		v := s.DurationMin
		out.DurationMin = &v
	}
	if s.Note != "" {
		v := s.Note
		out.Note = &v
	}
	return out
}

func strOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intOr0(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

func floatOr0(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func categoryOr(c *api.PlaceRefCategory) api.PlaceRefCategory {
	if c == nil {
		return ""
	}
	return *c
}
