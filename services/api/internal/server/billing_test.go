package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/billing"
)

type fakeBilling struct {
	got []billing.Event
	err error
}

func (f *fakeBilling) Apply(_ context.Context, e billing.Event) error {
	f.got = append(f.got, e)
	return f.err
}

func TestRevenueCatWebhook(t *testing.T) {
	verifier := fakeVerifier{token: "good", id: auth.Identity{UserID: "u_1"}}
	const payload = `{"event":{"id":"e1","type":"RENEWAL","app_user_id":"u1","product_id":"plus_monthly","event_timestamp_ms":1000,"expiration_at_ms":9000}}`
	post := func(h http.Handler, authz, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/revenuecat", strings.NewReader(body))
		if authz != "" {
			req.Header.Set("Authorization", authz)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	t.Run("applies a signed event", func(t *testing.T) {
		f := &fakeBilling{}
		rec := post(NewHandler(quiet(), verifier, WithBilling(f, "s3cret")), "Bearer s3cret", payload)
		if rec.Code != http.StatusNoContent || len(f.got) != 1 || f.got[0].UID != "u1" {
			t.Fatalf("status %d, events %+v", rec.Code, f.got)
		}
	})
	t.Run("rejects a missing or wrong secret", func(t *testing.T) {
		f := &fakeBilling{}
		h := NewHandler(quiet(), verifier, WithBilling(f, "s3cret"))
		for _, a := range []string{"", "Bearer nope", "s3cret"} {
			if rec := post(h, a, payload); rec.Code != http.StatusUnauthorized {
				t.Errorf("authz %q: status %d", a, rec.Code)
			}
		}
		if len(f.got) != 0 {
			t.Fatal("nothing may be applied")
		}
	})
	t.Run("bad body is 400 and a store failure is 500 so it is retried", func(t *testing.T) {
		h := NewHandler(quiet(), verifier, WithBilling(&fakeBilling{}, "s3cret"))
		if rec := post(h, "Bearer s3cret", `{"event":{}}`); rec.Code != http.StatusBadRequest {
			t.Errorf("bad body: %d", rec.Code)
		}
		h = NewHandler(quiet(), verifier, WithBilling(&fakeBilling{err: errors.New("down")}, "s3cret"))
		if rec := post(h, "Bearer s3cret", payload); rec.Code != http.StatusInternalServerError {
			t.Errorf("store failure: %d", rec.Code)
		}
	})
	t.Run("is off without a secret", func(t *testing.T) {
		if rec := post(NewHandler(quiet(), verifier, WithBilling(&fakeBilling{}, "")), "Bearer ", payload); rec.Code == http.StatusNoContent {
			t.Fatal("route must be off")
		}
	})
}
