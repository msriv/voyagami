package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/analytics"
	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/billing"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeEvents struct{ got map[string][]analytics.Record }

func (f *fakeEvents) Add(_ context.Context, rows []analytics.Record) error {
	for _, r := range rows {
		f.got[r.UID] = append(f.got[r.UID], r)
	}
	return nil
}

func TestSendEvents(t *testing.T) {
	store := &fakeEvents{got: map[string][]analytics.Record{}}
	consents := fakeConsents{}
	granted := model.ConsentGranted
	consents["u-yes"] = model.UnsetConsents().Apply(model.ConsentsPatch{Analytics: &granted}, time.Now())
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"yes": "u-yes", "no": "u-no"}, WithConsents(consents), WithEvents(store))
	now := time.Now().UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"events":[
		{"name":"trip_created","at":%q,"props":{"destinations":2,"source":"onboarding"}},
		{"name":"not_a_real_event","at":%q},
		{"name":"screen_viewed","at":%q,"props":{"phone":"+14155550123","route":"trip.home"}}],
		"context":{"sessionId":"s1","platform":"android","appVersion":"1.0.0+3"}}`, now, now, now)

	var r api.EventReceipt
	rec := call(h, "POST", "/v1/events", "yes", body)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.Accepted != 2 || r.Dropped != 1 || len(store.got["u-yes"]) != 2 {
		t.Fatalf("receipt %+v, stored %v", r, store.got)
	}
	if row := store.got["u-yes"][1]; row.Props != `{"route":"trip.home"}` || row.Platform != "android" || row.SessionID != "s1" || row.Source != "app" {
		t.Errorf("the phone number must not be stored, context must be: %+v", row)
	}

	// No consent, no events: nothing is stored, and the app is told they were dropped.
	_ = json.Unmarshal(call(h, "POST", "/v1/events", "no", body).Body.Bytes(), &r)
	if r.Accepted != 0 || r.Dropped != 3 || len(store.got["u-no"]) != 0 {
		t.Errorf("without consent: %+v, stored %v", r, store.got["u-no"])
	}

	if c := call(h, "POST", "/v1/events", "yes", `{"events":[]}`).Code; c != 400 {
		t.Errorf("empty batch: %d", c)
	}
	if c := call(h, "POST", "/v1/events", "", body).Code; c != 401 {
		t.Errorf("needs sign-in: %d", c)
	}
	off := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"yes": "u-yes"}, WithEvents(store))
	if c := call(off, "POST", "/v1/events", "yes", body).Code; c != 503 {
		t.Errorf("no consent store means events stay off: %d", c)
	}
}

func TestBillingEventNames(t *testing.T) {
	cases := []struct {
		ev   billing.Event
		want string
	}{
		{billing.Event{Type: "INITIAL_PURCHASE", PeriodType: "TRIAL", ProductID: "plus_month"}, "trial_started"},
		{billing.Event{Type: "INITIAL_PURCHASE", PeriodType: "NORMAL", ProductID: "plus_year"}, "subscription_started paid"},
		{billing.Event{Type: "INITIAL_PURCHASE", ProductID: "voyagami_trip_pass"}, "trip_pass_bought paid"},
		{billing.Event{Type: "RENEWAL", ProductID: "plus_month"}, "subscription_renewed paid"},
		{billing.Event{Type: "CANCELLATION", CancelReason: "CUSTOMER_SUPPORT"}, "purchase_refunded"},
		{billing.Event{Type: "CANCELLATION", CancelReason: "UNSUBSCRIBE"}, "subscription_cancelled"},
		{billing.Event{Type: "BILLING_ISSUE"}, ""},
	}
	for _, c := range cases {
		if got := strings.Join(billingEventNames(c.ev), " "); got != c.want {
			t.Errorf("%s/%s: got %q, want %q", c.ev.Type, c.ev.PeriodType, got, c.want)
		}
		for _, n := range billingEventNames(c.ev) {
			if !analytics.Names[n] {
				t.Errorf("%s is not on the allow-list", n)
			}
		}
	}
}

func TestServerEventWithoutConsentIsAnonymous(t *testing.T) {
	store := &fakeEvents{got: map[string][]analytics.Record{}}
	h := &handlers{productEvents: store, consents: fakeConsents{}}
	h.serverEvent(context.Background(), "u-no", "trial_started", map[string]any{"product": "plus_month"})
	if rows := store.got[""]; len(rows) != 1 || rows[0].Props != "" || rows[0].Source != "server" {
		t.Fatalf("counted without uid or props: %+v", store.got)
	}
}
