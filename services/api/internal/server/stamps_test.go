package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeStamps struct {
	mu     sync.Mutex
	earned map[string]map[model.StampKind]time.Time
	places map[string]int
}

func newFakeStamps() *fakeStamps {
	return &fakeStamps{earned: map[string]map[model.StampKind]time.Time{}, places: map[string]int{}}
}

func (f *fakeStamps) Award(_ context.Context, uid string, k model.StampKind, at time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.earned[uid] == nil {
		f.earned[uid] = map[model.StampKind]time.Time{}
	}
	if _, ok := f.earned[uid][k]; ok {
		return false, nil
	}
	f.earned[uid][k] = at
	return true, nil
}

func (f *fakeStamps) AddPlace(ctx context.Context, uid string, at time.Time) (bool, error) {
	f.mu.Lock()
	f.places[uid]++
	n := f.places[uid]
	f.mu.Unlock()
	if n == model.PlacesForStamp {
		return f.Award(ctx, uid, model.StampAddPlaces, at)
	}
	return false, nil
}

func (f *fakeStamps) List(_ context.Context, uid string) ([]model.EarnedStamp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []model.EarnedStamp
	for k, at := range f.earned[uid] {
		out = append(out, model.EarnedStamp{Kind: k, EarnedAt: at})
	}
	return out, nil
}

func TestStampsListAndCreateTrip(t *testing.T) {
	st := newFakeStamps()
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()), WithStamps(st))

	list := func() api.StampList {
		rec := call(h, "GET", "/v1/me/stamps", "owner", "")
		if rec.Code != 200 {
			t.Fatalf("list: %d %s", rec.Code, rec.Body)
		}
		var out api.StampList
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if got := list(); got.Total != 5 || got.Earned != 0 || len(got.Stamps) != 5 || got.Stamps[0].Kind != "createTrip" {
		t.Fatalf("fresh list: %+v", got)
	}
	if rec := call(h, "POST", "/v1/trips", "owner", newTripBody); rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	got := list()
	if got.Earned != 1 || !got.Stamps[0].Earned || got.Stamps[0].EarnedAt == nil || got.Stamps[1].Earned {
		t.Fatalf("after create trip: %+v", got)
	}
	// A second trip earns nothing new and keeps the first time.
	first := *got.Stamps[0].EarnedAt
	call(h, "POST", "/v1/trips", "owner", newTripBody)
	if again := list(); again.Earned != 1 || !again.Stamps[0].EarnedAt.Equal(first) {
		t.Fatalf("second trip changed the stamp: %+v", again)
	}
}

func TestPlacesStampNeedsThree(t *testing.T) {
	st := newFakeStamps()
	h := &handlers{stamps: st}
	for i := 1; i <= 4; i++ {
		h.countPlace(context.Background(), "u1")
		_, earned := st.earned["u1"][model.StampAddPlaces]
		if earned != (i >= model.PlacesForStamp) {
			t.Fatalf("after %d places earned=%v", i, earned)
		}
	}
}

func TestStampsOffAnswer503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u"})
	if got := call(h, "GET", "/v1/me/stamps", "owner", "").Code; got != 503 {
		t.Fatalf("got %d", got)
	}
}

type fakeRewarder struct{ granted []string }

func (f *fakeRewarder) GrantStampWeek(_ context.Context, uid string, _ time.Time) (bool, error) {
	f.granted = append(f.granted, uid)
	return true, nil
}

func TestFifthStampGrantsThePlusWeek(t *testing.T) {
	st, rw := newFakeStamps(), &fakeRewarder{}
	h := &handlers{stamps: st, stampReward: rw}
	ctx := context.Background()
	h.award(ctx, "u1", model.StampCreateTrip)
	h.award(ctx, "u1", model.StampInviteCrew)
	h.countPlace(ctx, "u1")
	h.countPlace(ctx, "u1")
	h.award(ctx, "u1", model.StampLogExpense)
	if len(rw.granted) != 0 {
		t.Fatalf("granted with four stamps: %v", rw.granted)
	}
	h.countPlace(ctx, "u1") // third place: the fourth stamp
	h.award(ctx, "u1", model.StampWriteJournal)
	if len(rw.granted) != 1 || rw.granted[0] != "u1" {
		t.Fatalf("want one grant for u1, got %v", rw.granted)
	}
	h.award(ctx, "u1", model.StampWriteJournal) // already earned: no second check
	if len(rw.granted) != 1 {
		t.Fatalf("granted twice: %v", rw.granted)
	}
}
