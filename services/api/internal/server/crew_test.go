package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// fakeInvites is an in-memory repo.Invites with the same rules as the real one.
type fakeInvites struct{ byID map[string]model.Invite }

func (f *fakeInvites) Create(_ context.Context, i model.Invite) error {
	if err := i.Validate(); err != nil {
		return err
	}
	f.byID[i.ID] = i
	return nil
}

func (f *fakeInvites) Get(_ context.Context, id string) (model.Invite, error) {
	i, ok := f.byID[id]
	if !ok {
		return model.Invite{}, repo.ErrNotFound
	}
	return i, nil
}

func (f *fakeInvites) List(_ context.Context, tripID string) ([]model.Invite, error) {
	var out []model.Invite
	for _, i := range f.byID {
		if i.TripID == tripID {
			out = append(out, i)
		}
	}
	return out, nil
}

func (f *fakeInvites) Revoke(_ context.Context, tripID, id string, now time.Time) error {
	i, ok := f.byID[id]
	if !ok || i.TripID != tripID {
		return repo.ErrNotFound
	}
	if i.RevokedAt.IsZero() {
		i.RevokedAt = now
		f.byID[id] = i
	}
	return nil
}

func (f *fakeInvites) Use(_ context.Context, id string, now time.Time) (model.Invite, error) {
	i, ok := f.byID[id]
	if !ok || !i.Usable(now) {
		return model.Invite{}, repo.ErrNotFound
	}
	i.UseCount++
	f.byID[id] = i
	return i, nil
}

func crewServer(trips *fakeTrips, invites *fakeInvites) http.Handler {
	v := tokenVerifier{"owner": "u-owner", "editor": "u-editor", "viewer": "u-viewer", "guest": "u-guest", "guest2": "u-guest2"}
	return NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithInvites(invites))
}

func newCrewTrip(t *testing.T, h http.Handler, trips *fakeTrips) string {
	t.Helper()
	rec := call(h, "POST", "/v1/trips", "owner", newTripBody)
	if rec.Code != 201 {
		t.Fatalf("create trip: %d %s", rec.Code, rec.Body)
	}
	var trip api.Trip
	if err := json.Unmarshal(rec.Body.Bytes(), &trip); err != nil {
		t.Fatal(err)
	}
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive, DisplayName: "Ed"}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive, DisplayName: "Vi"}
	return trip.Id
}

func createInvite(t *testing.T, h http.Handler, tok, tripID, body string) api.InviteCreated {
	t.Helper()
	rec := call(h, "POST", "/v1/trips/"+tripID+"/invites", tok, body)
	if rec.Code != 201 {
		t.Fatalf("create invite: %d %s", rec.Code, rec.Body)
	}
	var out api.InviteCreated
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestInviteJoinFlow(t *testing.T) {
	trips, invites := newFakeTrips(), &fakeInvites{byID: map[string]model.Invite{}}
	h := crewServer(trips, invites)
	id := newCrewTrip(t, h, trips)

	if rec := call(h, "POST", "/v1/trips/"+id+"/invites", "viewer", `{}`); rec.Code != 403 {
		t.Errorf("viewer creates: %d", rec.Code)
	}
	if rec := call(h, "POST", "/v1/trips/"+id+"/invites", "guest", `{}`); rec.Code != 404 {
		t.Errorf("stranger creates: %d", rec.Code)
	}
	if rec := call(h, "POST", "/v1/trips/"+id+"/invites", "owner", `{"maxUses":0}`); rec.Code != 400 {
		t.Errorf("maxUses 0: %d", rec.Code)
	}
	if rec := call(h, "POST", "/v1/trips/"+id+"/invites", "owner", `{"role":"owner"}`); rec.Code != 400 {
		t.Errorf("owner role: %d", rec.Code)
	}

	made := createInvite(t, h, "owner", id, `{"role":"viewer","maxUses":1}`)
	if len(made.Token) < 40 || made.Invite.Role != "viewer" || made.Invite.ExpiresAt == nil {
		t.Fatalf("invite: %+v", made)
	}
	if _, ok := invites.byID[made.Token]; ok {
		t.Error("token stored in the clear")
	}

	join := func(tok, token string) (int, api.Trip) {
		rec := call(h, "POST", "/v1/invites/join", tok, `{"token":"`+token+`"}`)
		var trip api.Trip
		_ = json.Unmarshal(rec.Body.Bytes(), &trip)
		return rec.Code, trip
	}
	code, trip := join("guest", made.Token)
	if code != 200 || trip.Id != id || trip.Role != "viewer" {
		t.Fatalf("join: %d %+v", code, trip)
	}
	if m := trips.members[id]["u-guest"]; m.Role != model.RoleViewer || m.DisplayName != fallbackName || m.InvitedVia != made.Invite.Id {
		t.Errorf("member: %+v", m)
	}
	// Joining again returns the trip and does not use up another seat.
	if code, _ := join("guest", made.Token); code != 200 || invites.byID[made.Invite.Id].UseCount != 1 {
		t.Errorf("rejoin: %d, uses %d", code, invites.byID[made.Invite.Id].UseCount)
	}
	// Single use: the next person is turned away.
	if code, _ := join("guest2", made.Token); code != 404 {
		t.Errorf("used up: %d", code)
	}
	if code, _ := join("guest2", "not-a-token"); code != 404 {
		t.Errorf("unknown token: %d", code)
	}
	if rec := call(h, "POST", "/v1/invites/join", "guest2", `{"token":""}`); rec.Code != 400 {
		t.Errorf("empty token: %d", rec.Code)
	}
}

func TestInviteExpiredAndRevoked(t *testing.T) {
	trips, invites := newFakeTrips(), &fakeInvites{byID: map[string]model.Invite{}}
	h := crewServer(trips, invites)
	id := newCrewTrip(t, h, trips)

	expired := createInvite(t, h, "owner", id, `{}`)
	i := invites.byID[expired.Invite.Id]
	i.ExpiresAt = time.Now().Add(-time.Minute)
	invites.byID[i.ID] = i
	if rec := call(h, "POST", "/v1/invites/join", "guest", `{"token":"`+expired.Token+`"}`); rec.Code != 404 {
		t.Errorf("expired: %d", rec.Code)
	}

	live := createInvite(t, h, "owner", id, `{"expiresInHours":2}`)
	if rec := call(h, "POST", "/v1/trips/"+id+"/invites", "editor", `{}`); rec.Code != 403 {
		t.Errorf("editor creates: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/v1/trips/"+id+"/invites/"+live.Invite.Id, "viewer", ""); rec.Code != 403 {
		t.Errorf("viewer revokes: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/v1/trips/"+id+"/invites/"+live.Invite.Id, "editor", ""); rec.Code != 403 {
		t.Errorf("editor revokes: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/v1/trips/"+id+"/invites/"+live.Invite.Id, "owner", ""); rec.Code != 204 {
		t.Errorf("owner revokes: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/v1/trips/"+id+"/invites/"+live.Invite.Id, "owner", ""); rec.Code != 204 {
		t.Errorf("revoke twice: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/v1/trips/"+id+"/invites/nope", "owner", ""); rec.Code != 404 {
		t.Errorf("unknown invite: %d", rec.Code)
	}
	if rec := call(h, "POST", "/v1/invites/join", "guest", `{"token":"`+live.Token+`"}`); rec.Code != 404 {
		t.Errorf("revoked: %d", rec.Code)
	}

	rec := call(h, "GET", "/v1/trips/"+id+"/invites", "owner", "")
	var list api.InviteList
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != 200 || len(list.Invites) != 2 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if rec := call(h, "GET", "/v1/trips/"+id+"/invites", "editor", ""); rec.Code != 403 {
		t.Errorf("editor lists: %d", rec.Code)
	}
}

func TestMembers(t *testing.T) {
	trips, invites := newFakeTrips(), &fakeInvites{byID: map[string]model.Invite{}}
	h := crewServer(trips, invites)
	id := newCrewTrip(t, h, trips)
	base := "/v1/trips/" + id + "/members"

	rec := call(h, "GET", base, "viewer", "")
	var list api.MemberList
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != 200 || len(list.Members) != 3 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if rec := call(h, "GET", base, "guest", ""); rec.Code != 404 {
		t.Errorf("stranger lists: %d", rec.Code)
	}

	if rec := call(h, "PATCH", base+"/u-viewer", "editor", `{"role":"editor"}`); rec.Code != 403 {
		t.Errorf("editor changes role: %d", rec.Code)
	}
	if rec := call(h, "PATCH", base+"/u-owner", "owner", `{"role":"viewer"}`); rec.Code != 403 {
		t.Errorf("owner demotes self: %d", rec.Code)
	}
	if rec := call(h, "PATCH", base+"/u-nobody", "owner", `{"role":"viewer"}`); rec.Code != 404 {
		t.Errorf("unknown member: %d", rec.Code)
	}
	if rec := call(h, "PATCH", base+"/u-viewer", "owner", `{"role":"owner"}`); rec.Code != 400 {
		t.Errorf("owner role: %d", rec.Code)
	}
	rec = call(h, "PATCH", base+"/u-viewer", "owner", `{"role":"editor"}`)
	var m api.Member
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil || rec.Code != 200 || m.Role != "editor" {
		t.Fatalf("set role: %d %s", rec.Code, rec.Body)
	}

	if rec := call(h, "DELETE", base+"/u-owner", "owner", ""); rec.Code != 403 {
		t.Errorf("owner leaves: %d", rec.Code)
	}
	if rec := call(h, "DELETE", base+"/u-viewer", "editor", ""); rec.Code != 403 {
		t.Errorf("editor removes another: %d", rec.Code)
	}
	if rec := call(h, "DELETE", base+"/u-guest", "owner", ""); rec.Code != 404 {
		t.Errorf("remove stranger: %d", rec.Code)
	}
	if rec := call(h, "DELETE", base+"/u-editor", "editor", ""); rec.Code != 204 || trips.members[id]["u-editor"].Status != model.MemberLeft {
		t.Errorf("leave: %d %+v", rec.Code, trips.members[id]["u-editor"])
	}
	if rec := call(h, "DELETE", base+"/u-viewer", "owner", ""); rec.Code != 204 || trips.members[id]["u-viewer"].Status != model.MemberRemoved {
		t.Errorf("remove: %d %+v", rec.Code, trips.members[id]["u-viewer"])
	}
	if rec := call(h, "GET", base, "viewer", ""); rec.Code != 404 {
		t.Errorf("removed member still reads: %d", rec.Code)
	}
	if rec := call(h, "GET", base, "owner", ""); rec.Code != 200 {
		t.Errorf("owner lists after: %d", rec.Code)
	}
}

func TestCrewRoutesOffWithoutStores(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()))
	if rec := call(h, "POST", "/v1/invites/join", "owner", `{"token":"x"}`); rec.Code != 503 {
		t.Errorf("join without invites: %d", rec.Code)
	}
}
