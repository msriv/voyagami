package server

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// BookingStore is the booking storage the routes need. repo.Bookings implements it.
type BookingStore = repo.Bookings

// WithBookings turns on the booking routes. They also need WithTrips.
func WithBookings(s BookingStore) Option { return func(h *handlers) { h.bookings = s } }

const msgNoBookings = "Bookings are not available right now."

func (h *handlers) bookingsOff() bool { return h.trips == nil || h.bookings == nil }

func placeOut(p *model.BookingPlace) *api.BookingPlace {
	if p == nil {
		return nil
	}
	out := api.BookingPlace{Name: p.Name}
	if p.Code != "" {
		out.Code = &p.Code
	}
	return &out
}

func placeIn(p *api.BookingPlace) *model.BookingPlace {
	if p == nil {
		return nil
	}
	out := model.BookingPlace{Name: strings.TrimSpace(p.Name)}
	if p.Code != nil {
		out.Code = strings.TrimSpace(*p.Code)
	}
	return &out
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func bookingFrom(b model.Booking) api.Booking {
	out := api.Booking{
		Id: b.ID, Kind: api.BookingKind(b.Kind), Title: b.Title, Provider: strPtr(b.Provider),
		StartTimeZone: strPtr(b.StartTimeZone), EndTimeZone: strPtr(b.EndTimeZone),
		From: placeOut(b.From), To: placeOut(b.To),
	}
	if !b.StartsAt.IsZero() {
		out.StartsAt = &b.StartsAt
	}
	if !b.EndsAt.IsZero() {
		out.EndsAt = &b.EndsAt
	}
	if b.Price != nil {
		out.Price = &api.Money{AmountMinor: b.Price.AmountMinor, Currency: b.Price.Currency}
	}
	return out
}

// bookingAccess checks the caller is on the trip, and when write is set that
// they are an owner or editor. deny is the 404 or 403 body to send, or nil.
func (h *handlers) bookingAccess(ctx context.Context, op, tripID string, write bool) (uid string, deny any, err error) {
	id, err := identity(ctx, op)
	if err != nil {
		return "", nil, err
	}
	role, err := h.myMembership(ctx, tripID, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return "", api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip or booking.")), nil
	}
	if err != nil {
		return "", nil, err
	}
	if write && role == model.RoleViewer {
		return "", errForbidden(ctx), nil
	}
	return id.UserID, nil, nil
}

// ListBookings returns a trip's bookings to any member.
func (h *handlers) ListBookings(ctx context.Context, req api.ListBookingsRequestObject) (api.ListBookingsResponseObject, error) {
	if h.bookingsOff() {
		return api.ListBookings503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoBookings))}, nil
	}
	_, deny, err := h.bookingAccess(ctx, "ListBookings", req.TripId, false)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ListBookings404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	list, err := h.bookings.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	out := api.BookingList{Bookings: make([]api.Booking, len(list))}
	for i, b := range list {
		out.Bookings[i] = bookingFrom(b)
	}
	return api.ListBookings200JSONResponse(out), nil
}

// CreateBooking adds a booking. Owners and editors only.
func (h *handlers) CreateBooking(ctx context.Context, req api.CreateBookingRequestObject) (api.CreateBookingResponseObject, error) {
	if h.bookingsOff() {
		return api.CreateBooking503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoBookings))}, nil
	}
	uid, deny, err := h.bookingAccess(ctx, "CreateBooking", req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.CreateBooking404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.CreateBooking403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	b := req.Body
	nb := model.Booking{
		ID: model.NewID(), Kind: string(b.Kind), Title: strings.TrimSpace(b.Title), CreatedBy: uid,
		From: placeIn(b.From), To: placeIn(b.To),
	}
	applyBookingFields(&nb, b.Provider, b.StartsAt, b.EndsAt, b.StartTimeZone, b.EndTimeZone, b.Price)
	saved, err := h.bookings.Create(ctx, req.TripId, nb)
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.CreateBooking400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.CreateBooking201JSONResponse(bookingFrom(saved)), nil
}

// applyBookingFields copies the optional fields that were sent onto b.
func applyBookingFields(b *model.Booking, provider *string, starts, ends *time.Time, startTZ, endTZ *string, price *api.Money) {
	if provider != nil {
		b.Provider = strings.TrimSpace(*provider)
	}
	if starts != nil {
		b.StartsAt = *starts
	}
	if ends != nil {
		b.EndsAt = *ends
	}
	if startTZ != nil {
		b.StartTimeZone = *startTZ
	}
	if endTZ != nil {
		b.EndTimeZone = *endTZ
	}
	if price != nil {
		b.Price = &model.Money{AmountMinor: price.AmountMinor, Currency: price.Currency}
	}
}

// UpdateBooking changes a booking. Owners and editors only.
func (h *handlers) UpdateBooking(ctx context.Context, req api.UpdateBookingRequestObject) (api.UpdateBookingResponseObject, error) {
	if h.bookingsOff() {
		return api.UpdateBooking503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoBookings))}, nil
	}
	_, deny, err := h.bookingAccess(ctx, "UpdateBooking", req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.UpdateBooking404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.UpdateBooking403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	cur, err := h.bookings.Get(ctx, req.TripId, req.BookingId)
	if errors.Is(err, repo.ErrNotFound) {
		return api.UpdateBooking404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such trip or booking."))}, nil
	}
	if err != nil {
		return nil, err
	}
	b := req.Body
	if b.Kind != nil {
		cur.Kind = string(*b.Kind)
	}
	if b.Title != nil {
		cur.Title = strings.TrimSpace(*b.Title)
	}
	if b.From != nil {
		cur.From = placeIn(b.From)
	}
	if b.To != nil {
		cur.To = placeIn(b.To)
	}
	applyBookingFields(&cur, b.Provider, b.StartsAt, b.EndsAt, b.StartTimeZone, b.EndTimeZone, b.Price)
	saved, err := h.bookings.Replace(ctx, req.TripId, cur)
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.UpdateBooking400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.UpdateBooking200JSONResponse(bookingFrom(saved)), nil
}

// DeleteBooking removes a booking. Owners and editors only.
func (h *handlers) DeleteBooking(ctx context.Context, req api.DeleteBookingRequestObject) (api.DeleteBookingResponseObject, error) {
	if h.bookingsOff() {
		return api.DeleteBooking503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoBookings))}, nil
	}
	_, deny, err := h.bookingAccess(ctx, "DeleteBooking", req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.DeleteBooking404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.DeleteBooking403JSONResponse{ForbiddenJSONResponse: d}, nil
	}
	if err := h.bookings.Delete(ctx, req.TripId, req.BookingId); err != nil {
		return nil, err
	}
	return api.DeleteBooking204Response{}, nil
}
