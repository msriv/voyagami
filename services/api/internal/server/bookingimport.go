package server

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/importer"
)

// WithImporter turns on reading bookings from pasted text and PDFs. It also
// needs WithBookings.
func WithImporter(p importer.Parser) Option { return func(h *handlers) { h.importer = p } }

const (
	msgNoImport  = "Reading bookings is not available right now."
	msgImportErr = "We couldn't read that right now. Try again, or add the booking by hand."
)

// ImportBookings asks the parser what bookings a document holds. Nothing is saved.
func (h *handlers) ImportBookings(ctx context.Context, req api.ImportBookingsRequestObject) (api.ImportBookingsResponseObject, error) {
	if h.bookingsOff() || h.importer == nil {
		return api.ImportBookings503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoImport))}, nil
	}
	_, deny, err := h.bookingAccess(ctx, "ImportBookings", req.TripId, true)
	if err != nil {
		return nil, err
	}
	switch d := deny.(type) {
	case api.NotFoundJSONResponse:
		return api.ImportBookings404JSONResponse{NotFoundJSONResponse: d}, nil
	case api.ForbiddenJSONResponse:
		return api.ImportBookings403JSONResponse{ForbiddenJSONResponse: d}, nil
	}

	b := req.Body
	var in importer.Input
	text := strings.TrimSpace(strOr(b.Text))
	pdf := strOr(b.PdfBase64)
	switch {
	case (text == "") == (pdf == ""):
		return api.ImportBookings400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "Send either text or a PDF.")}, nil
	case text != "":
		if utf8.RuneCountInString(text) > importer.MaxTextRunes {
			return api.ImportBookings400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "That text is too long.")}, nil
		}
		in.Text = text
	default:
		if base64.StdEncoding.DecodedLen(len(pdf)) > importer.MaxPDFBytes+4 {
			return api.ImportBookings400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "That PDF is too big.")}, nil
		}
		raw, err := base64.StdEncoding.DecodeString(pdf)
		if err != nil || len(raw) > importer.MaxPDFBytes || !strings.HasPrefix(string(raw), "%PDF") {
			return api.ImportBookings400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "That is not a PDF we can read.")}, nil
		}
		in.PDF = raw
	}

	cands, err := h.importer.Parse(ctx, in)
	out := api.BookingImportResult{Candidates: []api.BookingCandidate{}}
	switch {
	case errors.Is(err, importer.ErrNothingFound):
		return api.ImportBookings200JSONResponse(out), nil
	case err != nil:
		// The error holds no document text. Log it without the request body.
		slog.Default().Error("booking import failed", "requestId", requestIDFrom(ctx), "error", err)
		return api.ImportBookings503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgImportErr))}, nil
	}
	for _, c := range cands {
		bk := bookingFrom(c.Booking)
		out.Candidates = append(out.Candidates, api.BookingCandidate{
			Confidence: c.Confidence,
			Booking: api.BookingCreate{
				Kind: bk.Kind, Title: bk.Title, Provider: bk.Provider, StartsAt: bk.StartsAt, EndsAt: bk.EndsAt,
				StartTimeZone: bk.StartTimeZone, EndTimeZone: bk.EndTimeZone, From: bk.From, To: bk.To, Price: bk.Price,
			},
		})
	}
	return api.ImportBookings200JSONResponse(out), nil
}
