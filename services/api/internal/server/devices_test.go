package server

import (
	"context"
	"log/slog"
	"sort"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeDevices struct {
	by map[string]map[string]model.Device
}

func newFakeDevices() *fakeDevices { return &fakeDevices{by: map[string]map[string]model.Device{}} }

func (f *fakeDevices) Put(_ context.Context, uid string, d model.Device) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if f.by[uid] == nil {
		f.by[uid] = map[string]model.Device{}
	}
	f.by[uid][d.ID] = d
	return nil
}

func (f *fakeDevices) Delete(_ context.Context, uid, id string) error {
	delete(f.by[uid], id)
	return nil
}

func (f *fakeDevices) List(_ context.Context, uid string) ([]model.Device, error) {
	var out []model.Device
	for _, d := range f.by[uid] {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func TestDevices(t *testing.T) {
	store := newFakeDevices()
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"a": "u-a", "b": "u-b"}, WithDevices(store))

	if rec := call(h, "PUT", "/v1/me/devices/d1", "a", `{"platform":"ios","pushToken":"tok-1"}`); rec.Code != 204 {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	// Calling again replaces the token: one record per device.
	call(h, "PUT", "/v1/me/devices/d1", "a", `{"platform":"ios","pushToken":"tok-2"}`)
	if got := store.by["u-a"]["d1"].PushToken; got != "tok-2" || len(store.by["u-a"]) != 1 {
		t.Fatalf("not refreshed: %+v", store.by["u-a"])
	}
	// Tokens are per person.
	if len(store.by["u-b"]) != 0 {
		t.Fatal("leaked to another person")
	}
	for name, tc := range map[string]struct{ token, body string }{
		"bad platform": {"a", `{"platform":"web","pushToken":"x"}`},
		"no token":     {"a", `{"platform":"ios","pushToken":""}`},
	} {
		if rec := call(h, "PUT", "/v1/me/devices/d2", tc.token, tc.body); rec.Code != 400 {
			t.Errorf("%s: %d", name, rec.Code)
		}
	}
	if rec := call(h, "PUT", "/v1/me/devices/d1", "", `{"platform":"ios","pushToken":"x"}`); rec.Code != 401 {
		t.Errorf("signed out: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/v1/me/devices/d1", "a", ""); rec.Code != 204 || len(store.by["u-a"]) != 0 {
		t.Errorf("remove: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/v1/me/devices/d1", "a", ""); rec.Code != 204 {
		t.Errorf("remove twice: %d", rec.Code)
	}
}

func TestDevicesOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"a": "u-a"})
	if rec := call(h, "PUT", "/v1/me/devices/d1", "a", `{"platform":"ios","pushToken":"x"}`); rec.Code != 503 {
		t.Fatalf("got %d", rec.Code)
	}
}
