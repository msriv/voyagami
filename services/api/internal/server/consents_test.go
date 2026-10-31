package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeConsents map[string]model.Consents

func (f fakeConsents) Get(_ context.Context, uid string) (model.Consents, error) {
	if c, ok := f[uid]; ok {
		return c, nil
	}
	return model.UnsetConsents(), nil
}

func (f fakeConsents) Update(ctx context.Context, uid string, p model.ConsentsPatch, now time.Time) (model.Consents, error) {
	if err := p.Validate(); err != nil {
		return model.Consents{}, err
	}
	c, _ := f.Get(ctx, uid)
	f[uid] = c.Apply(p, now)
	return f[uid], nil
}

func TestConsents(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"a": "u-a", "b": "u-b"}, WithConsents(fakeConsents{}))
	var c api.Consents
	_ = json.Unmarshal(call(h, "GET", "/v1/me/consents", "a", "").Body.Bytes(), &c)
	if c.Location != api.ConsentState("unset") || c.Analytics != api.ConsentState("unset") || c.UpdatedAt != nil {
		t.Fatalf("new person: %+v", c)
	}
	rec := call(h, "PATCH", "/v1/me/consents", "a", `{"analytics":"granted","location":"denied"}`)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	if string(c.Analytics) != "granted" || string(c.Location) != "denied" || string(c.Photos) != "unset" || c.UpdatedAt == nil {
		t.Fatalf("after patch: %+v", c)
	}
	// A later patch keeps what was not sent, and a choice can be changed back.
	_ = json.Unmarshal(call(h, "PATCH", "/v1/me/consents", "a", `{"analytics":"denied"}`).Body.Bytes(), &c)
	if string(c.Analytics) != "denied" || string(c.Location) != "denied" {
		t.Fatalf("second patch: %+v", c)
	}
	// Another person's choices are their own.
	_ = json.Unmarshal(call(h, "GET", "/v1/me/consents", "b", "").Body.Bytes(), &c)
	if string(c.Analytics) != "unset" {
		t.Fatalf("other person: %+v", c)
	}
	if code := call(h, "PATCH", "/v1/me/consents", "a", `{"analytics":"unset"}`).Code; code != 400 {
		t.Errorf("unset cannot be sent: %d", code)
	}
	if code := call(h, "GET", "/v1/me/consents", "", "").Code; code != 401 {
		t.Errorf("needs sign-in: %d", code)
	}
}
