package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/analytics"
	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeEvents struct{ got map[string][]analytics.Event }

func (f *fakeEvents) Add(_ context.Context, uid string, ev []analytics.Event, _ time.Time) error {
	f.got[uid] = append(f.got[uid], ev...)
	return nil
}

func TestSendEvents(t *testing.T) {
	store := &fakeEvents{got: map[string][]analytics.Event{}}
	consents := fakeConsents{}
	granted := model.ConsentGranted
	consents["u-yes"] = model.UnsetConsents().Apply(model.ConsentsPatch{Analytics: &granted}, time.Now())
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"yes": "u-yes", "no": "u-no"}, WithConsents(consents), WithEvents(store))
	now := time.Now().UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"events":[
		{"name":"trip_created","at":%q,"props":{"destinations":2,"source":"onboarding"}},
		{"name":"not_a_real_event","at":%q},
		{"name":"screen_viewed","at":%q,"props":{"phone":"+14155550123","route":"trip.home"}}]}`, now, now, now)

	var r api.EventReceipt
	rec := call(h, "POST", "/v1/events", "yes", body)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if r.Accepted != 2 || r.Dropped != 1 || len(store.got["u-yes"]) != 2 {
		t.Fatalf("receipt %+v, stored %v", r, store.got)
	}
	if p := store.got["u-yes"][1].Props; len(p) != 1 || p["route"] != "trip.home" {
		t.Errorf("the phone number must not be stored: %v", p)
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
