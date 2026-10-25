package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type fakeMeets struct{ byID map[string]model.MeetPoint }

func (f *fakeMeets) Create(_ context.Context, _ string, m model.MeetPoint) (model.MeetPoint, error) {
	if err := m.Validate(); err != nil {
		return model.MeetPoint{}, err
	}
	m.CreatedAt = time.Now()
	f.byID[m.ID] = m
	return m, nil
}

func (f *fakeMeets) Get(_ context.Context, _, id string) (model.MeetPoint, error) {
	m, ok := f.byID[id]
	if !ok {
		return model.MeetPoint{}, repo.ErrNotFound
	}
	return m, nil
}

func (f *fakeMeets) List(context.Context, string) ([]model.MeetPoint, error) {
	var out []model.MeetPoint
	for _, m := range f.byID {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeMeets) Update(_ context.Context, _, id string, p model.MeetPointPatch) (model.MeetPoint, error) {
	if err := p.Validate(); err != nil {
		return model.MeetPoint{}, err
	}
	m, ok := f.byID[id]
	if !ok {
		return model.MeetPoint{}, repo.ErrNotFound
	}
	if p.Name != nil {
		m.Name = *p.Name
	}
	if p.Place != nil {
		m.Place = *p.Place
	}
	if p.At != nil {
		m.At = *p.At
	}
	f.byID[id] = m
	return m, nil
}

func (f *fakeMeets) SetConfirmed(_ context.Context, _, id, uid string, coming bool) (model.MeetPoint, error) {
	m, ok := f.byID[id]
	if !ok {
		return model.MeetPoint{}, repo.ErrNotFound
	}
	m.ConfirmedBy = slices.DeleteFunc(slices.Clone(m.ConfirmedBy), func(u string) bool { return u == uid })
	if coming {
		m.ConfirmedBy = append(m.ConfirmedBy, uid)
	}
	f.byID[id] = m
	return m, nil
}

func (f *fakeMeets) Delete(_ context.Context, _, id string) error {
	delete(f.byID, id)
	return nil
}

type fakePresence struct{ byUID map[string]model.Presence }

func (f *fakePresence) Put(_ context.Context, _ string, p model.Presence) error {
	if err := p.Validate(); err != nil {
		return err
	}
	f.byUID[p.UID] = p
	return nil
}

func (f *fakePresence) Get(_ context.Context, _, uid string) (model.Presence, error) {
	p, ok := f.byUID[uid]
	if !ok {
		return model.Presence{}, repo.ErrNotFound
	}
	return p, nil
}

func (f *fakePresence) Delete(_ context.Context, _, uid string) error {
	delete(f.byUID, uid)
	return nil
}

func (f *fakePresence) List(_ context.Context, _ string, now time.Time) ([]model.Presence, error) {
	var out []model.Presence
	for uid, p := range f.byUID {
		if !now.Before(p.EndsAt) {
			delete(f.byUID, uid)
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UID < out[j].UID })
	return out, nil
}

func meetServer(t *testing.T) (*fakeTrips, *fakePresence, *fakeMeets, *fakePlans, func(method, target, tok, body string) (int, []byte), string) {
	t.Helper()
	trips, pres, meets, plans := newFakeTrips(), &fakePresence{byUID: map[string]model.Presence{}}, &fakeMeets{byID: map[string]model.MeetPoint{}}, &fakePlans{}
	v := tokenVerifier{"owner": "u-owner", "editor": "u-editor", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithMeetPoints(meets, pres), WithEntitlements(entitlements.New(entitlements.Free()), plans))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	do := func(method, target, tok, body string) (int, []byte) {
		rec := call(h, method, target, tok, body)
		return rec.Code, rec.Body.Bytes()
	}
	return trips, pres, meets, plans, do, "/v1/trips/" + trip.Id
}

func TestMeetPoints(t *testing.T) {
	_, _, meets, _, do, base := meetServer(t)
	create := `{"name":"Lake gate","place":{"name":"Hoan Kiem Lake","lat":21.028,"lng":105.852},"at":"` + time.Now().Add(2*time.Hour).UTC().Format(time.RFC3339) + `"}`

	if c, _ := do("POST", base+"/meet-points", "viewer", create); c != 403 {
		t.Errorf("viewer creates: %d", c)
	}
	if c, _ := do("POST", base+"/meet-points", "other", create); c != 404 {
		t.Errorf("stranger creates: %d", c)
	}
	if c, _ := do("POST", base+"/meet-points", "editor", `{"name":"","place":{"name":"x","lat":1,"lng":1}}`); c != 400 {
		t.Errorf("empty name: %d", c)
	}
	if c, _ := do("POST", base+"/meet-points", "editor", `{"name":"No place"}`); c != 400 {
		t.Errorf("no place: %d", c)
	}
	c, body := do("POST", base+"/meet-points", "editor", create)
	var m api.MeetPoint
	if err := json.Unmarshal(body, &m); err != nil || c != 201 || m.Status != "active" || m.At == nil || m.IAmComing || len(m.ConfirmedBy) != 0 {
		t.Fatalf("create: %d %s", c, body)
	}
	one := base + "/meet-points/" + m.Id

	// Anyone on the trip can say they are coming; saying it twice counts once.
	for _, tok := range []string{"viewer", "viewer", "editor"} {
		if c, _ := do("PUT", one+"/confirmation", tok, ""); c != 200 {
			t.Fatalf("confirm %s: %d", tok, c)
		}
	}
	_, body = do("PUT", one+"/confirmation", "viewer", "")
	_ = json.Unmarshal(body, &m)
	if len(m.ConfirmedBy) != 2 || !m.IAmComing {
		t.Errorf("confirmed: %+v", m)
	}
	c, body = do("DELETE", one+"/confirmation", "viewer", "")
	_ = json.Unmarshal(body, &m)
	if c != 200 || len(m.ConfirmedBy) != 1 || m.IAmComing {
		t.Errorf("unconfirm: %d %+v", c, m)
	}
	if c, _ := do("PUT", base+"/meet-points/nope/confirmation", "viewer", ""); c != 404 {
		t.Errorf("confirm unknown: %d", c)
	}
	if c, _ := do("PUT", one+"/confirmation", "other", ""); c != 404 {
		t.Errorf("stranger confirms: %d", c)
	}

	if c, _ := do("PATCH", one, "viewer", `{"name":"x"}`); c != 403 {
		t.Errorf("viewer edits: %d", c)
	}
	c, body = do("PATCH", one, "owner", `{"name":"North gate"}`)
	_ = json.Unmarshal(body, &m)
	if c != 200 || m.Name != "North gate" || m.Place.Name == nil {
		t.Errorf("patch: %d %s", c, body)
	}
	if c, _ := do("PATCH", base+"/meet-points/nope", "owner", `{"name":"x"}`); c != 404 {
		t.Errorf("patch unknown: %d", c)
	}

	// A meet reads as ended once its time and three hours have passed.
	old := meets.byID[m.Id]
	old.At = time.Now().Add(-4 * time.Hour)
	meets.byID[m.Id] = old
	var list api.MeetPointList
	c, body = do("GET", base+"/meet-points", "viewer", "")
	if err := json.Unmarshal(body, &list); err != nil || c != 200 || len(list.MeetPoints) != 1 || list.MeetPoints[0].Status != "ended" {
		t.Fatalf("list: %d %s", c, body)
	}
	if c, _ := do("DELETE", one, "viewer", ""); c != 403 {
		t.Errorf("viewer deletes: %d", c)
	}
	if c, _ := do("DELETE", one, "editor", ""); c != 204 {
		t.Errorf("delete: %d", c)
	}
	if c, _ := do("DELETE", one, "editor", ""); c != 204 {
		t.Errorf("delete twice: %d", c)
	}
}

func TestSharingEndsByItself(t *testing.T) {
	_, pres, meets, plans, do, base := meetServer(t)
	loc := base + "/locations"

	if c, _ := do("PUT", loc+"/me", "other", `{"lat":21,"lng":105}`); c != 404 {
		t.Errorf("stranger shares: %d", c)
	}
	if c, _ := do("PUT", loc+"/me", "viewer", `{"lat":99,"lng":105}`); c != 400 {
		t.Errorf("bad coordinates: %d", c)
	}
	if c, _ := do("PUT", loc+"/me", "viewer", `{"lat":21,"lng":105,"durationMinutes":0}`); c != 400 {
		t.Errorf("zero duration: %d", c)
	}
	// Viewers can share. The default is one hour.
	c, body := do("PUT", loc+"/me", "viewer", `{"lat":21.03,"lng":105.85,"accuracyM":12}`)
	var mine api.SharedLocation
	if err := json.Unmarshal(body, &mine); err != nil || c != 200 || mine.Uid != "u-viewer" || mine.EndsAt.Sub(mine.UpdatedAt) != time.Hour {
		t.Fatalf("share: %d %s", c, body)
	}
	// Moving the dot keeps the end time: updates never stretch sharing.
	c, body = do("PUT", loc+"/me", "viewer", `{"lat":21.04,"lng":105.86}`)
	var moved api.SharedLocation
	_ = json.Unmarshal(body, &moved)
	if c != 200 || !moved.EndsAt.Equal(mine.EndsAt) || moved.Lat != 21.04 {
		t.Errorf("move: %d %s", c, body)
	}
	// More than an hour is Plus only; a trip pass unlocks it.
	if c, _ := do("PUT", loc+"/me", "viewer", `{"lat":21,"lng":105,"durationMinutes":240}`); c != 403 {
		t.Errorf("free, four hours: %d", c)
	}
	plans.trip = entitlements.Trip{Pass: true}
	c, body = do("PUT", loc+"/me", "viewer", `{"lat":21,"lng":105,"durationMinutes":240}`)
	_ = json.Unmarshal(body, &moved)
	if c != 200 || moved.EndsAt.Sub(moved.UpdatedAt) != 4*time.Hour {
		t.Errorf("with pass: %d %s", c, body)
	}

	// Sharing for a meet stops when the meet ends.
	_, mb := do("POST", base+"/meet-points", "owner", `{"name":"Gate","place":{"name":"Gate","lat":21,"lng":105},"at":"`+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+`"}`)
	var m api.MeetPoint
	_ = json.Unmarshal(mb, &m)
	c, body = do("PUT", loc+"/me", "editor", `{"lat":21,"lng":105,"durationMinutes":60,"meetPointId":"`+m.Id+`"}`)
	var ed api.SharedLocation
	_ = json.Unmarshal(body, &ed)
	if c != 200 || ed.MeetPointId == nil || *ed.MeetPointId != m.Id {
		t.Fatalf("share for meet: %d %s", c, body)
	}
	if c, _ := do("PUT", loc+"/me", "editor", `{"lat":21,"lng":105,"meetPointId":"nope"}`); c != 400 {
		t.Errorf("unknown meet: %d", c)
	}
	old := meets.byID[m.Id]
	old.At = time.Now().Add(-5 * time.Hour)
	meets.byID[m.Id] = old
	if c, _ := do("PUT", loc+"/me", "editor", `{"lat":21,"lng":105,"meetPointId":"`+m.Id+`"}`); c != 409 {
		t.Errorf("share for ended meet: %d", c)
	}

	var list api.SharedLocationList
	c, body = do("GET", loc, "owner", "")
	if err := json.Unmarshal(body, &list); err != nil || c != 200 || len(list.Locations) != 2 {
		t.Fatalf("list: %d %s", c, body)
	}
	// Once an entry's time is up it is gone from the list, and removed.
	e := pres.byUID["u-viewer"]
	e.EndsAt = time.Now().Add(-time.Second)
	pres.byUID["u-viewer"] = e
	c, body = do("GET", loc, "owner", "")
	_ = json.Unmarshal(body, &list)
	if c != 200 || len(list.Locations) != 1 || list.Locations[0].Uid != "u-editor" || len(pres.byUID) != 1 {
		t.Errorf("after end: %d %s", c, body)
	}
	// Starting again after an end gets a fresh hour.
	if c, body := do("PUT", loc+"/me", "viewer", `{"lat":21,"lng":105}`); c != 200 {
		t.Errorf("restart: %d %s", c, body)
	}
	if c, _ := do("DELETE", loc+"/me", "viewer", ""); c != 204 || pres.byUID["u-viewer"].UID != "" {
		t.Errorf("stop: %d", c)
	}
	if c, _ := do("DELETE", loc+"/me", "viewer", ""); c != 204 {
		t.Errorf("stop twice: %d", c)
	}
}
