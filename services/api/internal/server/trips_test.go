package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// fakeTrips is an in-memory TripStore that applies the same validation as the
// real one.
type fakeTrips struct {
	trips   map[string]model.Trip
	members map[string]map[string]model.Member
}

func newFakeTrips() *fakeTrips {
	return &fakeTrips{trips: map[string]model.Trip{}, members: map[string]map[string]model.Member{}}
}

func (f *fakeTrips) Create(_ context.Context, t model.Trip, owner model.Member) (model.Trip, error) {
	owner.Role, owner.Status = model.RoleOwner, model.MemberActive
	t.OwnerUID, t.MemberUIDs = owner.UID, []string{owner.UID}
	if err := t.Validate(); err != nil {
		return model.Trip{}, err
	}
	f.trips[t.ID] = t
	f.members[t.ID] = map[string]model.Member{owner.UID: owner}
	return t, nil
}

func (f *fakeTrips) Get(_ context.Context, id string) (model.Trip, error) {
	t, ok := f.trips[id]
	if !ok {
		return model.Trip{}, repo.ErrNotFound
	}
	return t, nil
}

func (f *fakeTrips) ListForMember(_ context.Context, uid string) ([]model.Trip, error) {
	var out []model.Trip
	for id, t := range f.trips {
		if _, ok := f.members[id][uid]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeTrips) Update(_ context.Context, id string, p model.TripPatch) (model.Trip, error) {
	if err := p.Validate(); err != nil {
		return model.Trip{}, err
	}
	t := f.trips[id]
	if p.Title != nil {
		t.Title = *p.Title
	}
	if p.Status != nil {
		t.Status = *p.Status
	}
	f.trips[id] = t
	return t, nil
}

func (f *fakeTrips) Member(_ context.Context, tripID, uid string) (model.Member, error) {
	m, ok := f.members[tripID][uid]
	if !ok {
		return model.Member{}, repo.ErrNotFound
	}
	return m, nil
}

type tokenVerifier map[string]string

func (v tokenVerifier) Verify(_ context.Context, token string) (auth.Identity, error) {
	if uid, ok := v[token]; ok {
		return auth.Identity{UserID: uid}, nil
	}
	return auth.Identity{}, auth.ErrInvalidToken
}

const newTripBody = `{"title":"Hanoi","destinations":[{"name":"Hanoi","countryCode":"VN","timeZone":"Asia/Ho_Chi_Minh"}],` +
	`"dates":{"start":"2027-02-01","end":"2027-02-05"},"timeZone":"Asia/Ho_Chi_Minh","settlementCurrency":"VND"}`

func call(h http.Handler, method, target, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func tripsServer(store *fakeTrips) http.Handler {
	v := tokenVerifier{"owner": "u-owner", "other": "u-other", "viewer": "u-viewer", "editor": "u-editor"}
	return NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(store))
}

func TestTripsLifecycleAndAccess(t *testing.T) {
	store := newFakeTrips()
	h := tripsServer(store)

	rec := call(h, "POST", "/v1/trips", "owner", newTripBody)
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var trip api.Trip
	if err := json.Unmarshal(rec.Body.Bytes(), &trip); err != nil {
		t.Fatal(err)
	}
	if trip.OwnerId != "u-owner" || trip.Role != "owner" || trip.Status != "planned" || trip.SettlementCurrency != "VND" {
		t.Fatalf("unexpected trip %+v", trip)
	}
	// Add an editor and a viewer directly in the store.
	store.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	store.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	path := "/v1/trips/" + trip.Id

	tests := []struct {
		name, method, token, body string
		want                      int
	}{
		{"owner reads", "GET", "owner", "", 200},
		{"viewer reads", "GET", "viewer", "", 200},
		{"non-member cannot read", "GET", "other", "", 404},
		{"unknown trip", "GET", "owner", "", 200},
		{"signed out", "GET", "", "", 401},
		{"editor renames", "PATCH", "editor", `{"title":"Hanoi 2027"}`, 200},
		{"viewer cannot edit", "PATCH", "viewer", `{"title":"x"}`, 403},
		{"non-member cannot edit", "PATCH", "other", `{"title":"x"}`, 404},
		{"editor cannot archive", "PATCH", "editor", `{"status":"past"}`, 403},
		{"owner archives", "PATCH", "owner", `{"status":"past"}`, 200},
		{"blank title is rejected", "PATCH", "owner", `{"title":"  "}`, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := path
			if tt.name == "unknown trip" {
				p, tt.want = "/v1/trips/nope", 404
			}
			if got := call(h, tt.method, p, tt.token, tt.body).Code; got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}

	var list api.TripList
	rec = call(h, "GET", "/v1/trips", "owner", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Trips) != 1 || list.Trips[0].Status != "past" || list.Trips[0].Title != "Hanoi 2027" {
		t.Fatalf("owner list: %s", rec.Body)
	}
	rec = call(h, "GET", "/v1/trips", "other", "")
	if !strings.Contains(rec.Body.String(), `"trips":[]`) {
		t.Fatalf("non-member should see an empty list, got %s", rec.Body)
	}
}

func TestListSkipsTripsAMemberLeft(t *testing.T) {
	store := newFakeTrips()
	h := tripsServer(store)
	rec := call(h, "POST", "/v1/trips", "owner", newTripBody)
	var trip api.Trip
	_ = json.Unmarshal(rec.Body.Bytes(), &trip)
	store.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberLeft}
	if got := call(h, "GET", "/v1/trips/"+trip.Id, "editor", "").Code; got != 404 {
		t.Fatalf("member who left should get 404, got %d", got)
	}
	if body := call(h, "GET", "/v1/trips", "editor", "").Body.String(); !strings.Contains(body, `"trips":[]`) {
		t.Fatalf("member who left should see no trips, got %s", body)
	}
}

func TestCreateTripValidation(t *testing.T) {
	h := tripsServer(newFakeTrips())
	bad := strings.Replace(newTripBody, `"settlementCurrency":"VND"`, `"settlementCurrency":"vnd"`, 1)
	rec := call(h, "POST", "/v1/trips", "owner", bad)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "settlementCurrency") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestTripsOffAnswer503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u"})
	if got := call(h, "GET", "/v1/trips", "owner", "").Code; got != 503 {
		t.Fatalf("got %d", got)
	}
}
