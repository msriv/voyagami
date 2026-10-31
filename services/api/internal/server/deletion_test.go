package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type fakeDeletions struct{ by map[string]model.Deletion }

func (f *fakeDeletions) Schedule(_ context.Context, uid string, now time.Time) (model.Deletion, error) {
	if d, ok := f.by[uid]; ok {
		return d, nil
	}
	d := model.Deletion{UserID: uid, RequestedAt: now, DeleteOn: now.Add(model.DeletionGrace)}
	f.by[uid] = d
	return d, nil
}
func (f *fakeDeletions) Get(_ context.Context, uid string) (model.Deletion, error) {
	if d, ok := f.by[uid]; ok {
		return d, nil
	}
	return model.Deletion{}, repo.ErrNotFound
}
func (f *fakeDeletions) Cancel(_ context.Context, uid string) error { delete(f.by, uid); return nil }
func (f *fakeDeletions) Due(context.Context, time.Time, int) ([]string, error) {
	return nil, nil
}
func (f *fakeDeletions) Purge(context.Context, string, time.Time) error { return nil }

func TestAccountDeletion(t *testing.T) {
	f := &fakeDeletions{by: map[string]model.Deletion{}}
	renewing := fakePlans{user: entitlements.User{Plus: true, Sub: &entitlements.Subscription{WillRenew: true}}}
	cancelled := fakePlans{user: entitlements.User{Plus: true, Sub: &entitlements.Subscription{WillRenew: false}}}
	v := tokenVerifier{"owner": "u-owner"}
	free := NewHandler(slog.New(slog.DiscardHandler), v, WithDeletions(f), WithEntitlements(entitlements.New(entitlements.Free()), fakePlans{}))

	var d api.Deletion
	_ = json.Unmarshal(call(free, "GET", "/v1/me/deletion", "owner", "").Body.Bytes(), &d)
	if d.Scheduled {
		t.Fatalf("nothing scheduled yet: %+v", d)
	}
	rec := call(free, "DELETE", "/v1/me", "owner", "")
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &d)
	if !d.Scheduled || d.DeleteOn == nil || time.Until(*d.DeleteOn) < 29*24*time.Hour {
		t.Fatalf("scheduled: %+v", d)
	}
	first := *d.DeleteOn
	// Asking again keeps the first date.
	_ = json.Unmarshal(call(free, "DELETE", "/v1/me", "owner", "").Body.Bytes(), &d)
	if !d.DeleteOn.Equal(first) {
		t.Errorf("second ask moved the date: %v vs %v", d.DeleteOn, first)
	}
	if c := call(free, "DELETE", "/v1/me/deletion", "owner", "").Code; c != 204 {
		t.Errorf("cancel: %d", c)
	}
	_ = json.Unmarshal(call(free, "GET", "/v1/me/deletion", "owner", "").Body.Bytes(), &d)
	if d.Scheduled {
		t.Errorf("still scheduled after cancel: %+v", d)
	}

	// A renewing subscription blocks it with a plain code; a cancelled one does not.
	blocked := NewHandler(slog.New(slog.DiscardHandler), v, WithDeletions(f), WithEntitlements(entitlements.New(entitlements.Free()), renewing))
	rec = call(blocked, "DELETE", "/v1/me", "owner", "")
	var e api.Error
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if rec.Code != 409 || e.Code != "subscription_active" {
		t.Fatalf("renewing: %d %s", rec.Code, rec.Body)
	}
	if _, ok := f.by["u-owner"]; ok {
		t.Error("a blocked request must not be recorded")
	}
	ok := NewHandler(slog.New(slog.DiscardHandler), v, WithDeletions(f), WithEntitlements(entitlements.New(entitlements.Free()), cancelled))
	if c := call(ok, "DELETE", "/v1/me", "owner", "").Code; c != 200 {
		t.Errorf("cancelled subscription: %d", c)
	}

	if c := call(free, "DELETE", "/v1/me", "", "").Code; c != 401 {
		t.Errorf("needs sign-in: %d", c)
	}
	off := NewHandler(slog.New(slog.DiscardHandler), v)
	if c := call(off, "DELETE", "/v1/me", "owner", "").Code; c != 503 {
		t.Errorf("off: %d", c)
	}
}
