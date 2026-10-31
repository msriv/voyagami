package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type fakeSponsored struct {
	cards  []model.SponsoredCard
	hidden map[string]time.Time
	seen   []string
}

func (f *fakeSponsored) Live(_ context.Context, now time.Time) ([]model.SponsoredCard, error) {
	var out []model.SponsoredCard
	for _, c := range f.cards {
		if c.LiveAt(now) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeSponsored) Get(_ context.Context, id string) (model.SponsoredCard, error) {
	for _, c := range f.cards {
		if c.ID == id {
			return c, nil
		}
	}
	return model.SponsoredCard{}, repo.ErrNotFound
}

func (f *fakeSponsored) Hidden(_ context.Context, _ string, now time.Time) (map[string]bool, error) {
	out := map[string]bool{}
	for id, until := range f.hidden {
		if until.After(now) {
			out[id] = true
		}
	}
	return out, nil
}

func (f *fakeSponsored) Hide(_ context.Context, _, id string, until time.Time) error {
	f.hidden[id] = until
	return nil
}

func (f *fakeSponsored) Record(_ context.Context, _, id string, kind model.SponsoredEvent, _ time.Time) error {
	f.seen = append(f.seen, id+":"+string(kind))
	return nil
}

func TestSponsoredCard(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	f := &fakeSponsored{hidden: map[string]time.Time{}, cards: []model.SponsoredCard{
		{ID: "a", Sponsor: "A", Title: "Low", URL: "https://a.example", Priority: 1, StartsAt: past},
		{ID: "b", Sponsor: "B", Title: "High", URL: "https://b.example", Priority: 5, StartsAt: past},
		{ID: "c", Sponsor: "C", Title: "Japan only", URL: "https://c.example", Priority: 9, Countries: []string{"JP"}, StartsAt: past},
		{ID: "d", Sponsor: "D", Title: "Not yet", URL: "https://d.example", Priority: 9, StartsAt: time.Now().Add(time.Hour)},
	}}
	trips := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithSponsored(f))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	path := "/v1/trips/" + trip.Id + "/sponsored"

	get := func() api.SponsoredResult {
		rec := call(h, "GET", path, "owner", "")
		if rec.Code != 200 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		var r api.SponsoredResult
		_ = json.Unmarshal(rec.Body.Bytes(), &r)
		return r
	}
	if r := get(); r.Card == nil || r.Card.Id != "b" {
		t.Fatalf("want b (highest priority that fits a Vietnam trip), got %+v", r.Card)
	}

	rec := call(h, "POST", "/v1/sponsored/b/hide", "owner", "")
	if rec.Code != 200 {
		t.Fatalf("hide: %d %s", rec.Code, rec.Body)
	}
	var hid api.SponsoredHidden
	_ = json.Unmarshal(rec.Body.Bytes(), &hid)
	if d := time.Until(hid.HiddenUntil); d < 29*24*time.Hour || d > 31*24*time.Hour {
		t.Errorf("hidden for %v, want about 30 days", d)
	}
	if r := get(); r.Card == nil || r.Card.Id != "a" {
		t.Errorf("after hiding b want a, got %+v", r.Card)
	}

	if c := call(h, "POST", "/v1/sponsored/a/impressions", "owner", "").Code; c != 204 {
		t.Errorf("impression: %d", c)
	}
	if c := call(h, "POST", "/v1/sponsored/a/clicks", "owner", "").Code; c != 204 {
		t.Errorf("click: %d", c)
	}
	if len(f.seen) != 2 || f.seen[0] != "a:impression" || f.seen[1] != "a:click" {
		t.Errorf("recorded %v", f.seen)
	}
	if c := call(h, "POST", "/v1/sponsored/nope/clicks", "owner", "").Code; c != 404 {
		t.Errorf("unknown card: %d", c)
	}
	if c := call(h, "GET", path, "other", "").Code; c != 404 {
		t.Errorf("non-member: %d", c)
	}
	if c := call(h, "GET", path, "", "").Code; c != 401 {
		t.Errorf("signed out: %d", c)
	}
}

func TestSponsoredOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	if c := call(h, "POST", "/v1/sponsored/x/clicks", "owner", "").Code; c != 503 {
		t.Errorf("%d", c)
	}
}
