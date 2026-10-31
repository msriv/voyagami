package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/flightstatus"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithFlightStatus turns on live flight status. It also needs WithBookings.
// Without it the route answers 503.
func WithFlightStatus(p flightstatus.Provider) Option { return func(h *handlers) { h.flights = p } }

const (
	msgNoFlights  = "Flight status is not available right now."
	msgFlightsErr = "We couldn't check that flight right now. Try again later."
)

// GetFlightStatus asks the provider about a flight booking.
func (h *handlers) GetFlightStatus(ctx context.Context, req api.GetFlightStatusRequestObject) (api.GetFlightStatusResponseObject, error) {
	if h.bookingsOff() || h.flights == nil {
		return api.GetFlightStatus503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoFlights))}, nil
	}
	_, deny, err := h.bookingAccess(ctx, "GetFlightStatus", req.TripId, false)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.GetFlightStatus404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	b, err := h.bookings.Get(ctx, req.TripId, req.BookingId)
	notFound := api.GetFlightStatus404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such flight."))}
	if errors.Is(err, repo.ErrNotFound) {
		return notFound, nil
	} else if err != nil {
		return nil, err
	}
	f, ok := flightstatus.Parse(b.Title, b.StartsAt)
	if b.Kind != "flight" || !ok {
		return notFound, nil
	}
	st, err := h.flights.Status(ctx, f)
	if err != nil {
		slog.Default().Error("flight status failed", "requestId", requestIDFrom(ctx), "error", err)
		return api.GetFlightStatus503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgFlightsErr))}, nil
	}
	out := api.FlightStatus{State: api.FlightStatusState(st.State), Departure: legOut(st.Departure), Arrival: legOut(st.Arrival)}
	if !st.UpdatedAt.IsZero() {
		out.UpdatedAt = &st.UpdatedAt
	}
	return api.GetFlightStatus200JSONResponse(out), nil
}

func legOut(l flightstatus.Leg) *api.FlightLeg {
	var o api.FlightLeg
	set := func(t time.Time) *time.Time {
		if t.IsZero() {
			return nil
		}
		return &t
	}
	o.Scheduled, o.Estimated = set(l.Scheduled), set(l.Estimated)
	if l.Terminal != "" {
		o.Terminal = &l.Terminal
	}
	if l.Gate != "" {
		o.Gate = &l.Gate
	}
	return &o
}
