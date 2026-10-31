package firestorerepo

import (
	"errors"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestDeletionSchedule(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewDeletions(c)
	uid := model.NewID()
	now := time.Now()
	if _, err := s.Get(ctx, uid); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("none yet: %v", err)
	}
	d, err := s.Schedule(ctx, uid, now)
	if err != nil || d.DeleteOn.Sub(now) < 29*24*time.Hour {
		t.Fatalf("schedule: %+v %v", d, err)
	}
	again, _ := s.Schedule(ctx, uid, now.Add(time.Hour))
	if !again.DeleteOn.Equal(d.DeleteOn) {
		t.Errorf("asking again moved the date: %v vs %v", again.DeleteOn, d.DeleteOn)
	}
	if due, _ := s.Due(ctx, now.Add(time.Hour), 100); contains(due, uid) {
		t.Error("not due after one hour")
	}
	if due, _ := s.Due(ctx, now.Add(31*24*time.Hour), 100); !contains(due, uid) {
		t.Error("due after 31 days")
	}
	if err := s.Cancel(ctx, uid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, uid); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("after cancel: %v", err)
	}
	if err := s.Cancel(ctx, uid); err != nil {
		t.Errorf("cancelling none: %v", err)
	}
}

func TestPurge(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewDeletions(c)
	trips := NewTrips(c)
	now := time.Now()

	leaver, friend := model.NewID(), model.NewID()
	put := func(path string, doc map[string]any) {
		t.Helper()
		if _, err := c.Doc(path).Set(ctx, doc); err != nil {
			t.Fatal(err)
		}
	}
	put("users/"+leaver, map[string]any{"displayName": "Lee"})
	put("users/"+leaver+"/devices/d1", map[string]any{"pushToken": "tok"})
	put("consents/"+leaver, map[string]any{"analytics": "granted"})
	put("entitlements/"+leaver, map[string]any{"plus": map[string]any{"active": false}})
	put("sessions/s-"+leaver, map[string]any{"userId": leaver})
	put("sessions/s-"+friend, map[string]any{"userId": friend})
	put("phoneIndex/h-"+leaver, map[string]any{"uid": leaver})

	// A trip only the leaver is on goes away with everything under it.
	solo, err := trips.Create(ctx, sampleTrip("2026-12-01", "2026-12-05"), member(leaver, model.RoleOwner))
	if err != nil {
		t.Fatal(err)
	}
	put("trips/"+solo.ID+"/stops/s1", map[string]any{"name": "Lake"})
	put("trips/"+solo.ID+"/journal/2026-12-01_"+leaver, map[string]any{"authorUid": leaver, "text": "private"})

	// A shared trip the leaver owns stays, with the friend as the new owner.
	shared, err := trips.Create(ctx, sampleTrip("2026-12-10", "2026-12-15"), member(leaver, model.RoleOwner))
	if err != nil {
		t.Fatal(err)
	}
	fm := member(friend, model.RoleEditor)
	if err := trips.AddMember(ctx, shared.ID, fm); err != nil {
		t.Fatal(err)
	}
	put("trips/"+shared.ID+"/expenses/e1", map[string]any{"description": "Dinner", "paidByUid": leaver})
	put("trips/"+shared.ID+"/journal/2026-12-10_"+leaver, map[string]any{"authorUid": leaver, "text": "private"})
	put("trips/"+shared.ID+"/journal/2026-12-10_"+friend, map[string]any{"authorUid": friend, "text": "friend's"})
	put("trips/"+shared.ID+"/presence/"+leaver, map[string]any{"uid": leaver})

	if _, err := s.Schedule(ctx, leaver, now.Add(-31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // twice: a rerun must be harmless
		if err := s.Purge(ctx, leaver, now); err != nil {
			t.Fatalf("purge %d: %v", i+1, err)
		}
	}

	exists := func(path string) bool {
		snap, err := c.Doc(path).Get(ctx)
		return err == nil && snap.Exists()
	}
	for _, gone := range []string{
		"users/" + leaver, "users/" + leaver + "/devices/d1", "consents/" + leaver, "entitlements/" + leaver,
		"sessions/s-" + leaver, "phoneIndex/h-" + leaver, "trips/" + solo.ID, "trips/" + solo.ID + "/stops/s1",
		"trips/" + shared.ID + "/journal/2026-12-10_" + leaver, "trips/" + shared.ID + "/presence/" + leaver,
	} {
		if exists(gone) {
			t.Errorf("%s should be deleted", gone)
		}
	}
	for _, kept := range []string{"sessions/s-" + friend, "trips/" + shared.ID, "trips/" + shared.ID + "/expenses/e1", "trips/" + shared.ID + "/journal/2026-12-10_" + friend} {
		if !exists(kept) {
			t.Errorf("%s should stay", kept)
		}
	}

	got, err := trips.Get(ctx, shared.ID)
	if err != nil || got.OwnerUID != friend || len(got.MemberUIDs) != 1 || got.MemberUIDs[0] != friend {
		t.Errorf("shared trip after purge: %+v %v", got, err)
	}
	m, err := trips.Member(ctx, shared.ID, leaver)
	if err != nil || m.Status != model.MemberLeft || m.DisplayName != "" {
		t.Errorf("leaver's member record: %+v %v", m, err)
	}
	if fm2, _ := trips.Member(ctx, shared.ID, friend); fm2.Role != model.RoleOwner {
		t.Errorf("friend's role: %v", fm2.Role)
	}
	if _, err := s.Get(ctx, leaver); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("a purged request is no longer pending: %v", err)
	}
	if due, _ := s.Due(ctx, now, 100); contains(due, leaver) {
		t.Error("a purged account is not due again")
	}
}
