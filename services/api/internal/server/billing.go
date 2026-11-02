package server

import (
	"crypto/subtle"
	"io"
	"log/slog"
	"net/http"

	"github.com/msriv/voyagami/services/api/internal/billing"
)

// WithBilling turns on POST /v1/billing/revenuecat, the RevenueCat webhook.
// RevenueCat sends the secret as the Authorization header (the value is set in
// its dashboard), not a sign-in token, so the route sits outside the OpenAPI
// contract and the sign-in check. An empty secret leaves the route off.
func WithBilling(s billing.Store, secret string) Option {
	return func(h *handlers) { h.billing, h.billingSecret = s, secret }
}

const maxWebhookBytes = 64 << 10

func (h *handlers) mountBilling(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/billing/revenuecat", func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, []byte("Bearer "+h.billingSecret)) != 1 {
			writeError(w, r, http.StatusUnauthorized, codeUnauthenticated, "Sign-in is required.")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBytes))
		if err != nil {
			writeError(w, r, http.StatusBadRequest, codeBadRequest, "The event could not be read.")
			return
		}
		ev, err := billing.Parse(body)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, codeBadRequest, "The event could not be read.")
			return
		}
		// A failed write answers 500 so RevenueCat sends the event again.
		if err := h.billing.Apply(r.Context(), ev); err != nil {
			slog.Default().Error("billing event failed", "requestId", requestIDFrom(r.Context()), "type", ev.Type, "eventId", ev.ID, "error", err)
			writeError(w, r, http.StatusInternalServerError, codeInternal, "Something went wrong.")
			return
		}
		uid := ev.UID
		if ev.Anonymous() {
			uid = ""
		}
		for _, name := range billingEventNames(ev) {
			h.serverEvent(r.Context(), uid, name, billingProps(ev))
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// billingEventNames maps a store notification to analytics events (VOY-224).
// "paid" marks any money taken; the funnel counts each person's first one.
func billingEventNames(ev billing.Event) []string {
	switch ev.Type {
	case "INITIAL_PURCHASE":
		switch {
		case ev.IsTripPass():
			return []string{"trip_pass_bought", "paid"}
		case ev.PeriodType == "TRIAL":
			return []string{"trial_started"}
		default:
			return []string{"subscription_started", "paid"}
		}
	case "RENEWAL":
		return []string{"subscription_renewed", "paid"}
	case "CANCELLATION":
		if ev.Refunded() {
			return []string{"purchase_refunded"}
		}
		return []string{"subscription_cancelled"}
	}
	return nil
}

func billingProps(ev billing.Event) map[string]any {
	plan := "plus"
	if ev.IsTripPass() {
		plan = "pass"
	}
	p := map[string]any{"product": ev.ProductID, "plan": plan}
	if ev.TripID != "" {
		p["tripId"] = ev.TripID
	}
	return p
}
