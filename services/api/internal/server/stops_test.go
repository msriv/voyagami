package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// fakeStops is an in-memory StopStore.
type fakeStops struct{ stops map[string]model.Stop }

func (f *fakeStops) Create(_ context.Context, s model.Stop) (model.Stop, error) {
	if err := s.Validate(); err != nil {
		return model.Stop{}, err
	}
	f.stops[s.ID] = s
	return s, nil
}

func (f *fakeStops) Get(_ context.Context, _, id string) (model.Stop, error) {
	s, ok := f.stops[id]
	if !ok {
		return model.Stop{}, repo.ErrNotFound
	}
	return s, nil
}

func (f *fakeStops) List(context.Context, string) ([]model.Stop, error) {
	var out []model.Stop
	for _, s := range f.stops {
		if s.DeletedAt == nil {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].Order < out[j].Order
	})
	return out, nil
}

func (f *fakeStops) Update(_ context.Context, _, id string, p model.StopPatch) (model.Stop, error) {
	if err := p.Validate(); err != nil {
		return model.Stop{}, err
	}
	s, ok := f.stops[id]
	if !ok {
		return model.Stop{}, repo.ErrNotFound
	}
	if p.Date != nil {
		s.Date = *p.Date
	}
	if p.Order != nil {
		s.Order = *p.Order
	}
	if p.StartLocal != nil {
		s.StartLocal = *p.StartLocal
	}
	if p.DurationMin != nil {
		s.DurationMin = *p.DurationMin
	}
	if p.Note != nil {
		s.Note = *p.Note
	}
	if p.Deleted {
		now := time.Now()
		s.DeletedAt = &now
	}
	f.stops[id] = s
	return s, nil
}

type stopRig struct {
	h      http.Handler
	trips  *fakeTrips
	tripID string
	t      *testing.T
}

func newStopRig(t *testing.T) *stopRig {
	t.Helper()
	trips := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner", "editor": "u-editor", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithStops(&fakeStops{stops: map[string]model.Stop{}}))
	var trip api.Trip
	rec := call(h, "POST", "/v1/trips", "owner", newTripBody)
	if err := json.Unmarshal(rec.Body.Bytes(), &trip); err != nil {
		t.Fatal(err)
	}
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	return &stopRig{h: h, trips: trips, tripID: trip.Id, t: t}
}

func (r *stopRig) path(suffix string) string { return "/v1/trips/" + r.tripID + "/stops" + suffix }

func (r *stopRig) add(token, date, name string, extra string) (int, api.Stop) {
	body := `{"date":"` + date + `","place":{"name":"` + name + `","lat":21.03,"lng":105.85}` + extra + `}`
	rec := call(r.h, "POST", r.path(""), token, body)
	var s api.Stop
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	return rec.Code, s
}

// day lists the names on a day in the order the API returns them.
func (r *stopRig) day(date string) []string {
	rec := call(r.h, "GET", r.path(""), "owner", "")
	var l api.StopList
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil || rec.Code != 200 {
		r.t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var names []string
	for _, s := range l.Stops {
		if s.Date == date {
			names = append(names, *s.Place.Name)
		}
	}
	return names
}

func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (r *stopRig) expect(date string, want ...string) {
	r.t.Helper()
	if got := r.day(date); !same(got, want) {
		r.t.Fatalf("%s: got %v, want %v", date, got, want)
	}
}

func TestStopsOrderAndMoveBetweenDays(t *testing.T) {
	r := newStopRig(t)
	const d1, d2 = "2027-02-01", "2027-02-02"

	_, a := r.add("owner", d1, "A", "")
	_, b := r.add("editor", d1, "B", "")
	_, c := r.add("owner", d1, "C", "")
	r.expect(d1, "A", "B", "C")

	// Insert before an existing stop, including before the first.
	r.add("owner", d1, "D", `,"beforeStopId":"`+c.Id+`"`)
	r.add("owner", d1, "E", `,"beforeStopId":"`+a.Id+`"`)
	r.expect(d1, "E", "A", "B", "D", "C")

	patch := func(id, body string) int { return call(r.h, "PATCH", r.path("/"+id), "owner", body).Code }

	// Reorder within the day writes only the moved stop.
	if code := patch(c.Id, `{"beforeStopId":"`+a.Id+`"}`); code != 200 {
		t.Fatalf("reorder: %d", code)
	}
	r.expect(d1, "E", "C", "A", "B", "D")

	// Move to another day: lands at the end, then before an existing stop.
	if code := patch(b.Id, `{"date":"`+d2+`"}`); code != 200 {
		t.Fatalf("move: %d", code)
	}
	r.expect(d1, "E", "C", "A", "D")
	r.expect(d2, "B")
	if code := patch(a.Id, `{"date":"`+d2+`","beforeStopId":"`+b.Id+`"}`); code != 200 {
		t.Fatalf("move before: %d", code)
	}
	r.expect(d1, "E", "C", "D")
	r.expect(d2, "A", "B")

	// A stop cannot go before itself, or before a stop on another day.
	if code := patch(a.Id, `{"beforeStopId":"`+a.Id+`"}`); code != 400 {
		t.Errorf("before itself: %d", code)
	}
	if code := patch(a.Id, `{"beforeStopId":"`+c.Id+`"}`); code != 400 {
		t.Errorf("before a stop on another day: %d", code)
	}
	r.expect(d2, "A", "B")
}

func TestStopsFieldsDeleteAndErrors(t *testing.T) {
	r := newStopRig(t)
	const day = "2027-02-03"
	code, s := r.add("owner", day, "Cafe", `,"startLocal":"09:30","durationMin":45,"note":"egg coffee"`)
	if code != 201 || *s.StartLocal != "09:30" || *s.DurationMin != 45 || s.TimeZone != "Asia/Ho_Chi_Minh" {
		t.Fatalf("create: %d %+v", code, s)
	}

	// An empty value clears a field.
	rec := call(r.h, "PATCH", r.path("/"+s.Id), "editor", `{"startLocal":"","durationMin":0,"note":"x"}`)
	var got api.Stop
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != 200 || got.StartLocal != nil || got.DurationMin != nil || *got.Note != "x" {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body)
	}

	tests := []struct {
		name, method, path, token, body string
		want                            int
	}{
		{"viewer reads", "GET", r.path(""), "viewer", "", 200},
		{"non-member cannot read", "GET", r.path(""), "other", "", 404},
		{"signed out", "GET", r.path(""), "", "", 401},
		{"viewer cannot add", "POST", r.path(""), "viewer", `{"date":"` + day + `","place":{"placeId":"p1"}}`, 403},
		{"non-member cannot add", "POST", r.path(""), "other", `{"date":"` + day + `","place":{"placeId":"p1"}}`, 404},
		{"viewer cannot edit", "PATCH", r.path("/" + s.Id), "viewer", `{"note":"y"}`, 403},
		{"viewer cannot delete", "DELETE", r.path("/" + s.Id), "viewer", "", 403},
		{"day outside the trip", "POST", r.path(""), "owner", `{"date":"2027-03-01","place":{"placeId":"p1"}}`, 400},
		{"bad date", "POST", r.path(""), "owner", `{"date":"3 Feb","place":{"placeId":"p1"}}`, 400},
		{"pin needs a name", "POST", r.path(""), "owner", `{"date":"` + day + `","place":{"lat":1,"lng":1}}`, 400},
		{"pin needs real coordinates", "POST", r.path(""), "owner", `{"date":"` + day + `","place":{"name":"x","lat":91,"lng":1}}`, 400},
		{"bad time", "PATCH", r.path("/" + s.Id), "owner", `{"startLocal":"25:00"}`, 400},
		{"move outside the trip", "PATCH", r.path("/" + s.Id), "owner", `{"date":"2030-01-01"}`, 400},
		{"unknown stop", "PATCH", r.path("/nope"), "owner", `{"note":"y"}`, 404},
		{"catalogue place", "POST", r.path(""), "owner", `{"date":"` + day + `","place":{"placeId":"hanoi-1"}}`, 201},
		{"owner deletes", "DELETE", r.path("/" + s.Id), "owner", "", 204},
		{"deleted stop is gone", "DELETE", r.path("/" + s.Id), "owner", "", 404},
		{"deleted stop cannot be edited", "PATCH", r.path("/" + s.Id), "owner", `{"note":"z"}`, 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := call(r.h, tt.method, tt.path, tt.token, tt.body).Code; got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStopsOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()))
	if got := call(h, "GET", "/v1/trips/t/stops", "owner", "").Code; got != 503 {
		t.Fatalf("got %d, want 503", got)
	}
}

func (f *fakeStops) SetOrders(_ context.Context, _ string, orders map[string]string) error {
	for id, o := range orders {
		s, ok := f.stops[id]
		if !ok {
			return repo.ErrNotFound
		}
		s.Order = o
		f.stops[id] = s
	}
	return nil
}
