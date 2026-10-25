package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type fakeComments struct {
	byID map[string]model.Comment
	n    int
}

func (f *fakeComments) Create(_ context.Context, _ string, c model.Comment) (model.Comment, error) {
	if err := c.Validate(); err != nil {
		return model.Comment{}, err
	}
	f.n++
	c.CreatedAt = time.Unix(int64(f.n), 0)
	f.byID[c.ID] = c
	return c, nil
}

func (f *fakeComments) Get(_ context.Context, _, id string) (model.Comment, error) {
	c, ok := f.byID[id]
	if !ok {
		return model.Comment{}, repo.ErrNotFound
	}
	return c, nil
}

func (f *fakeComments) List(context.Context, string) ([]model.Comment, error) {
	var out []model.Comment
	for _, c := range f.byID {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (f *fakeComments) Edit(_ context.Context, _, id, body string, mentions []string, now time.Time) (model.Comment, error) {
	c, ok := f.byID[id]
	if !ok {
		return model.Comment{}, repo.ErrNotFound
	}
	c.Body, c.Mentions, c.EditedAt = body, mentions, now
	f.byID[id] = c
	return c, nil
}

func (f *fakeComments) Delete(_ context.Context, _, id string) error {
	delete(f.byID, id)
	return nil
}

func TestComments(t *testing.T) {
	trips, stops := newFakeTrips(), &fakeStops{stops: map[string]model.Stop{}}
	v := tokenVerifier{"owner": "u-owner", "editor": "u-editor", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithStops(stops), WithComments(&fakeComments{byID: map[string]model.Comment{}}))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	stops.stops["s1"] = model.Stop{ID: "s1", TripID: trip.Id, Date: "2027-02-02", Order: "a0", Place: model.PlaceRef{PlaceID: "p1"}, TimeZone: "Asia/Ho_Chi_Minh"}
	base := "/v1/trips/" + trip.Id + "/comments"
	post := func(tok, body string) (int, api.Comment) {
		rec := call(h, "POST", base, tok, body)
		var c api.Comment
		_ = json.Unmarshal(rec.Body.Bytes(), &c)
		return rec.Code, c
	}

	// Every member can comment, viewers too; strangers cannot.
	code, c1 := post("viewer", `{"targetType":"stop","targetId":"s1","body":"Looks great.","mentions":["u-editor"]}`)
	if code != 201 || c1.AuthorUid != "u-viewer" || len(c1.Mentions) != 1 || c1.EditedAt != nil {
		t.Fatalf("viewer comments: %d %+v", code, c1)
	}
	if code, _ := post("other", `{"targetType":"stop","targetId":"s1","body":"hi"}`); code != 404 {
		t.Errorf("stranger comments: %d", code)
	}
	for name, body := range map[string]string{
		"empty":           `{"targetType":"stop","targetId":"s1","body":"  "}`,
		"unknown stop":    `{"targetType":"stop","targetId":"nope","body":"x"}`,
		"wrong trip id":   `{"targetType":"trip","targetId":"nope","body":"x"}`,
		"stranger tagged": `{"targetType":"stop","targetId":"s1","body":"x","mentions":["u-other"]}`,
		"double tagged":   `{"targetType":"stop","targetId":"s1","body":"x","mentions":["u-editor","u-editor"]}`,
	} {
		if code, _ := post("editor", body); code != 400 {
			t.Errorf("%s: %d", name, code)
		}
	}
	code, c2 := post("editor", `{"targetType":"trip","targetId":"`+trip.Id+`","body":"Who books the taxi?"}`)
	if code != 201 {
		t.Fatalf("trip comment: %d", code)
	}

	var list api.CommentList
	rec := call(h, "GET", base+"?targetType=stop&targetId=s1", "viewer", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != 200 || len(list.Comments) != 1 || list.Comments[0].Id != c1.Id {
		t.Fatalf("thread: %d %s", rec.Code, rec.Body)
	}
	rec = call(h, "GET", base, "viewer", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Comments) != 2 || list.Comments[0].Id != c1.Id {
		t.Errorf("all, oldest first: %s", rec.Body)
	}
	if rec := call(h, "GET", base+"?targetType=stop", "viewer", ""); rec.Code != 400 {
		t.Errorf("half a filter: %d", rec.Code)
	}

	// Only the author edits; the owner can delete anyone's, others only their own.
	one := base + "/" + c1.Id
	if rec := call(h, "PATCH", one, "editor", `{"body":"hijack"}`); rec.Code != 403 {
		t.Errorf("edit other's: %d", rec.Code)
	}
	if rec := call(h, "PATCH", one, "owner", `{"body":"hijack"}`); rec.Code != 403 {
		t.Errorf("owner edits other's: %d", rec.Code)
	}
	rec = call(h, "PATCH", one, "viewer", `{"body":"Looks great, edited."}`)
	var edited api.Comment
	if err := json.Unmarshal(rec.Body.Bytes(), &edited); err != nil || rec.Code != 200 || edited.Body != "Looks great, edited." || edited.EditedAt == nil || len(edited.Mentions) != 0 {
		t.Fatalf("edit: %d %s", rec.Code, rec.Body)
	}
	if rec := call(h, "PATCH", base+"/nope", "viewer", `{"body":"x"}`); rec.Code != 404 {
		t.Errorf("edit unknown: %d", rec.Code)
	}
	if rec := call(h, "DELETE", one, "editor", ""); rec.Code != 403 {
		t.Errorf("delete other's: %d", rec.Code)
	}
	if rec := call(h, "DELETE", one, "viewer", ""); rec.Code != 204 {
		t.Errorf("delete own: %d", rec.Code)
	}
	if rec := call(h, "DELETE", one, "viewer", ""); rec.Code != 204 {
		t.Errorf("delete twice: %d", rec.Code)
	}
	if rec := call(h, "DELETE", base+"/"+c2.Id, "owner", ""); rec.Code != 204 {
		t.Errorf("owner deletes: %d", rec.Code)
	}
}
