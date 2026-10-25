package server

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/rates"
)

func TestGetRate(t *testing.T) {
	v := tokenVerifier{"owner": "u-owner"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithRates(rates.NewFixed()))

	if rec := call(h, "GET", "/v1/rates?from=USD&to=VND", "", ""); rec.Code != 401 {
		t.Errorf("no token: %d", rec.Code)
	}
	rec := call(h, "GET", "/v1/rates?from=usd&to=VND", "owner", "")
	var q api.RateQuote
	if err := json.Unmarshal(rec.Body.Bytes(), &q); err != nil || rec.Code != 200 || q.Rate.Numerator != 250 || q.Rate.Denominator != 1 || q.From != "USD" || len(q.AsOf) != 10 {
		t.Fatalf("quote: %d %s", rec.Code, rec.Body)
	}
	for _, target := range []string{"/v1/rates?from=US&to=VND", "/v1/rates?from=USD&to=V1D"} {
		if rec := call(h, "GET", target, "owner", ""); rec.Code != 400 {
			t.Errorf("%s: %d", target, rec.Code)
		}
	}
	if rec := call(h, "GET", "/v1/rates?from=USD", "owner", ""); rec.Code != 400 {
		t.Errorf("missing to: %d", rec.Code)
	}
	if rec := call(h, "GET", "/v1/rates?from=USD&to=XXX", "owner", ""); rec.Code != 404 {
		t.Errorf("unknown pair: %d", rec.Code)
	}
	off := NewHandler(slog.New(slog.DiscardHandler), v)
	if rec := call(off, "GET", "/v1/rates?from=USD&to=VND", "owner", ""); rec.Code != 503 {
		t.Errorf("without provider: %d", rec.Code)
	}
}
