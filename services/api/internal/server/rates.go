package server

import (
	"context"
	"errors"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/rates"
)

// WithRates turns on the exchange rate route. Without it the route answers 503.
func WithRates(p rates.Provider) Option { return func(h *handlers) { h.rates = p } }

func isCurrencyCode(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range strings.ToUpper(s) {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// GetRate suggests the rate for an expense in another currency.
func (h *handlers) GetRate(ctx context.Context, req api.GetRateRequestObject) (api.GetRateResponseObject, error) {
	if _, err := identity(ctx, "GetRate"); err != nil {
		return nil, err
	}
	if h.rates == nil {
		return api.GetRate503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, "Exchange rates are not available right now."))}, nil
	}
	if !isCurrencyCode(req.Params.From) || !isCurrencyCode(req.Params.To) {
		return api.GetRate400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "from and to must be three-letter currency codes.")}, nil
	}
	q, err := h.rates.Quote(ctx, req.Params.From, req.Params.To)
	if errors.Is(err, rates.ErrNoRate) {
		return api.GetRate404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No rate for that pair of currencies. Enter one by hand."))}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetRate200JSONResponse{From: q.From, To: q.To, AsOf: q.AsOf, Rate: api.Rate{Numerator: q.Rate.Numerator, Denominator: q.Rate.Denominator}}, nil
}
