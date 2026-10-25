package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type fakeBookings struct{ byID map[string]model.Booking }

func (f *fakeBookings) Create(_ context.Context, _ string, b model.Booking) (model.Booking, error) {
	if err := b.Validate(); err != nil {
		return model.Booking{}, err
	}
	f.byID[b.ID] = b
	return b, nil
}

func (f *fakeBookings) Get(_ context.Context, _, id string) (model.Booking, error) {
	b, ok := f.byID[id]
	if !ok {
		return model.Booking{}, repo.ErrNotFound
	}
	return b, nil
}

func (f *fakeBookings) List(context.Context, string) ([]model.Booking, error) {
	var out []model.Booking
	for _, b := range f.byID {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	return out, nil
}

func (f *fakeBookings) Replace(_ context.Context, _ string, b model.Booking) (model.Booking, error) {
	if err := b.Validate(); err != nil {
		return model.Booking{}, err
	}
	f.byID[b.ID] = b
	return b, nil
}

func (f *fakeBookings) Delete(_ context.Context, _, id string) error {
	delete(f.byID, id)
	return nil
}

func TestBookings(t *testing.T) {
	trips := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner", "editor": "u-editor", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithBookings(&fakeBookings{byID: map[string]model.Booking{}}))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	path := "/v1/trips/" + trip.Id + "/bookings"

	flight := `{"kind":"flight","title":"BLR to HAN","provider":"VietJet","startsAt":"2027-02-01T02:30:00Z","endsAt":"2027-02-01T08:05:00Z",` +
		`"startTimeZone":"Asia/Kolkata","endTimeZone":"Asia/Ho_Chi_Minh","from":{"code":"BLR","name":"Bengaluru"},"to":{"code":"HAN","name":"Hanoi"},` +
		`"price":{"amountMinor":1250000,"currency":"INR"}}`
	rec := call(h, "POST", path, "editor", flight)
	var b api.Booking
	_ = json.Unmarshal(rec.Body.Bytes(), &b)
	if rec.Code != 201 || b.Kind != api.BookingKind("flight") || b.To == nil || *b.To.Code != "HAN" || b.Price.AmountMinor != 1250000 || b.StartsAt == nil {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}

	for name, c := range map[string]struct {
		token, body string
		want        int
	}{
		"viewer":        {"viewer", flight, 403},
		"outsider":      {"other", flight, 404},
		"bad kind":      {"owner", `{"kind":"boat","title":"x"}`, 400},
		"no title":      {"owner", `{"kind":"stay","title":" "}`, 400},
		"ends first":    {"owner", `{"kind":"stay","title":"x","startsAt":"2027-02-02T00:00:00Z","endsAt":"2027-02-01T00:00:00Z"}`, 400},
		"bad zone":      {"owner", `{"kind":"stay","title":"x","startTimeZone":"Mars/Base"}`, 400},
		"bad currency":  {"owner", `{"kind":"stay","title":"x","price":{"amountMinor":5,"currency":"inr"}}`, 400},
		"nameless from": {"owner", `{"kind":"train","title":"x","from":{"code":"HAN","name":""}}`, 400},
	} {
		if got := call(h, "POST", path, c.token, c.body).Code; got != c.want {
			t.Errorf("%s: got %d, want %d", name, got, c.want)
		}
	}

	one := path + "/" + b.Id
	rec = call(h, "PATCH", one, "owner", `{"title":"BLR to HAN direct","provider":""}`)
	var changed api.Booking
	_ = json.Unmarshal(rec.Body.Bytes(), &changed)
	if rec.Code != 200 || changed.Title != "BLR to HAN direct" || changed.Provider != nil || changed.From == nil || changed.StartsAt == nil {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	if c := call(h, "PATCH", one, "viewer", `{"title":"x"}`).Code; c != 403 {
		t.Errorf("viewer patch: %d", c)
	}
	if c := call(h, "PATCH", path+"/missing", "owner", `{"title":"x"}`).Code; c != 404 {
		t.Errorf("patch missing: %d", c)
	}
	if c := call(h, "PATCH", one, "owner", `{"endsAt":"2027-01-01T00:00:00Z"}`).Code; c != 400 {
		t.Errorf("end before start: %d", c)
	}

	var list api.BookingList
	_ = json.Unmarshal(call(h, "GET", path, "viewer", "").Body.Bytes(), &list)
	if len(list.Bookings) != 1 {
		t.Fatalf("list: %+v", list)
	}
	if c := call(h, "GET", path, "other", "").Code; c != 404 {
		t.Errorf("outsider list: %d", c)
	}
	if c := call(h, "DELETE", one, "viewer", "").Code; c != 403 {
		t.Errorf("viewer delete: %d", c)
	}
	for i := 0; i < 2; i++ {
		if c := call(h, "DELETE", one, "editor", "").Code; c != 204 {
			t.Errorf("delete %d: %d", i, c)
		}
	}
}

func TestBookingsOff(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	if c := call(h, "GET", "/v1/trips/t/bookings", "owner", "").Code; c != 503 {
		t.Errorf("got %d", c)
	}
}
