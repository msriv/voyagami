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

type fakeVideos struct{ jobs []model.RecapVideo }

func (f *fakeVideos) Enqueue(_ context.Context, v model.RecapVideo) (model.RecapVideo, bool, error) {
	for _, j := range f.jobs {
		if j.TripID == v.TripID && j.Active() {
			return j, false, nil
		}
	}
	v.ID, v.State = "j"+string(rune('1'+len(f.jobs))), model.RecapQueued
	f.jobs = append(f.jobs, v)
	return v, true, nil
}
func (f *fakeVideos) Latest(_ context.Context, tripID string) (model.RecapVideo, error) {
	for i := len(f.jobs) - 1; i >= 0; i-- {
		if f.jobs[i].TripID == tripID {
			return f.jobs[i], nil
		}
	}
	return model.RecapVideo{}, repo.ErrNotFound
}
func (f *fakeVideos) Claim(context.Context, time.Time, time.Time) (model.RecapVideo, bool, error) {
	return model.RecapVideo{}, false, nil
}
func (f *fakeVideos) Progress(context.Context, string, int) error      { return nil }
func (f *fakeVideos) Finish(context.Context, string, string) error     { return nil }
func (f *fakeVideos) Fail(context.Context, string, string, bool) error { return nil }

type fakeSigner struct{}

func (fakeSigner) Render(context.Context, model.RecapVideo, func(int)) (string, error) {
	return "", nil
}
func (fakeSigner) URL(_ context.Context, o string) (string, error) {
	return "https://files.example/" + o + "?sig=1", nil
}

func TestRecapVideo(t *testing.T) {
	store := &fakeVideos{}
	trips := newFakeTrips()
	plans := fakePlans{}
	v := tokenVerifier{"owner": "u-owner", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithRecapVideo(store, fakeSigner{}),
		WithEntitlements(entitlements.New(entitlements.Free()), plans))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	path := "/v1/trips/" + trip.Id + "/recap/video"

	if c := call(h, "GET", path, "owner", "").Code; c != 404 {
		t.Errorf("before any request: %d", c)
	}
	rec := call(h, "POST", path, "owner", "")
	var j api.RecapVideo
	_ = json.Unmarshal(rec.Body.Bytes(), &j)
	if rec.Code != 202 || j.State != "queued" || j.MaxSeconds != 30 || !j.Watermark {
		t.Fatalf("free trip: %d %s", rec.Code, rec.Body)
	}
	var again api.RecapVideo
	_ = json.Unmarshal(call(h, "POST", path, "owner", "").Body.Bytes(), &again)
	if again.Id != j.Id || len(store.jobs) != 1 {
		t.Errorf("a second request while one runs returns the same job: %s vs %s, %d jobs", again.Id, j.Id, len(store.jobs))
	}
	if c := call(h, "POST", path, "viewer", "").Code; c != 403 {
		t.Errorf("viewer: %d", c)
	}
	if c := call(h, "GET", path, "viewer", "").Code; c != 200 {
		t.Errorf("viewer may read: %d", c)
	}
	if c := call(h, "GET", path, "other", "").Code; c != 404 {
		t.Errorf("non-member: %d", c)
	}

	store.jobs[0].State, store.jobs[0].Object = model.RecapReady, "videos/j1.mp4"
	var ready api.RecapVideo
	_ = json.Unmarshal(call(h, "GET", path, "owner", "").Body.Bytes(), &ready)
	if ready.Url == nil || *ready.Url != "https://files.example/videos/j1.mp4?sig=1" {
		t.Errorf("ready job needs a download link: %+v", ready)
	}

	// Plus lifts the cap and the watermark.
	store.jobs[0].State = model.RecapFailed
	plans.user.Plus = true
	h2 := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithRecapVideo(store, fakeSigner{}),
		WithEntitlements(entitlements.New(entitlements.Free()), plans))
	var plus api.RecapVideo
	_ = json.Unmarshal(call(h2, "POST", path, "owner", "").Body.Bytes(), &plus)
	if plus.MaxSeconds != -1 || plus.Watermark {
		t.Errorf("Plus: %+v", plus)
	}
}

func TestRecapVideoOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()))
	if c := call(h, "POST", "/v1/trips/t/recap/video", "owner", "").Code; c != 503 {
		t.Errorf("%d", c)
	}
}
