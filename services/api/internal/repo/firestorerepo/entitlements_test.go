package firestorerepo

import (
	"testing"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestEntitlementsAccount(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewEntitlements(c)

	put := func(t *testing.T, doc map[string]any) string {
		t.Helper()
		uid := model.NewID()
		if _, err := c.Collection(entitlementsCollection).Doc(uid).Set(ctx, doc); err != nil {
			t.Fatal(err)
		}
		return uid
	}
	tests := []struct {
		name      string
		uid       func(t *testing.T) string
		wantPlus  bool
		wantDraft int
	}{
		{"no document is Free", func(*testing.T) string { return model.NewID() }, false, 0},
		{"active subscription", func(t *testing.T) string { return put(t, map[string]any{"plus": map[string]any{"active": true}}) }, true, 0},
		{"inactive subscription", func(t *testing.T) string { return put(t, map[string]any{"plus": map[string]any{"active": false}}) }, false, 0},
		{"running trial", func(t *testing.T) string {
			return put(t, map[string]any{"trial": map[string]any{"endsAt": time.Now().Add(24 * time.Hour)}})
		}, true, 0},
		{"ended trial", func(t *testing.T) string {
			return put(t, map[string]any{"trial": map[string]any{"endsAt": time.Now().Add(-time.Hour)}})
		}, false, 0},
		{"ask drafts are counted", func(t *testing.T) string {
			return put(t, map[string]any{"usage": map[string]any{"askDrafts": 2, "periodStart": monthStart(time.Now())}})
		}, false, 2},
		{"ask drafts from an earlier month read as zero", func(t *testing.T) string {
			return put(t, map[string]any{"usage": map[string]any{"askDrafts": 3, "periodStart": monthStart(time.Now()).AddDate(0, -1, 0)}})
		}, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plus, drafts, err := s.Account(ctx, tt.uid(t))
			if err != nil {
				t.Fatal(err)
			}
			if plus != tt.wantPlus || drafts != tt.wantDraft {
				t.Errorf("got plus=%v drafts=%d, want plus=%v drafts=%d", plus, drafts, tt.wantPlus, tt.wantDraft)
			}
		})
	}
}

func TestAskDraftsResetEachMonth(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewEntitlements(c)
	uid := model.NewID()
	jan := time.Date(2027, 1, 30, 10, 0, 0, 0, time.UTC)
	for want := 1; want <= 3; want++ {
		if got, err := s.AddAskDraft(ctx, uid, jan); err != nil || got != want {
			t.Fatalf("draft %d: got %d, %v", want, got, err)
		}
	}
	// A new month starts again from one and keeps what else is in the document.
	if _, err := c.Collection(entitlementsCollection).Doc(uid).Set(ctx, map[string]any{"plus": map[string]any{"active": true}}, firestore.MergeAll); err != nil {
		t.Fatal(err)
	}
	if got, err := s.AddAskDraft(ctx, uid, time.Date(2027, 2, 1, 0, 0, 1, 0, time.UTC)); err != nil || got != 1 {
		t.Fatalf("first draft of february: got %d, %v", got, err)
	}
	plus, _, err := s.Account(ctx, uid)
	if err != nil || !plus {
		t.Fatalf("plus lost: %v %v", plus, err)
	}
}

func TestGrantStampWeek(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewEntitlements(c)
	now := time.Now().Truncate(time.Second)

	t.Run("a new account gets a week once", func(t *testing.T) {
		uid := model.NewID()
		if ok, err := s.GrantStampWeek(ctx, uid, now); err != nil || !ok {
			t.Fatalf("first grant: %v %v", ok, err)
		}
		if plus, _, _ := s.Account(ctx, uid); !plus {
			t.Fatal("must have Plus during the week")
		}
		if ok, err := s.GrantStampWeek(ctx, uid, now); err != nil || ok {
			t.Fatalf("second grant: %v %v", ok, err)
		}
	})
	t.Run("a running trial is extended", func(t *testing.T) {
		uid := model.NewID()
		ends := now.Add(2 * 24 * time.Hour)
		if _, err := c.Collection(entitlementsCollection).Doc(uid).Set(ctx, map[string]any{"trial": map[string]any{"startedAt": now.Add(-5 * 24 * time.Hour), "endsAt": ends}}); err != nil {
			t.Fatal(err)
		}
		if ok, err := s.GrantStampWeek(ctx, uid, now); err != nil || !ok {
			t.Fatalf("grant: %v %v", ok, err)
		}
		snap, _ := c.Collection(entitlementsCollection).Doc(uid).Get(ctx)
		var d entitlementsDoc
		_ = snap.DataTo(&d)
		if want := ends.Add(StampWeek); !d.Trial.EndsAt.Equal(want) {
			t.Fatalf("trial ends %v, want %v", d.Trial.EndsAt, want)
		}
	})
	t.Run("a paid subscriber gets nothing", func(t *testing.T) {
		uid := model.NewID()
		if _, err := c.Collection(entitlementsCollection).Doc(uid).Set(ctx, map[string]any{"plus": map[string]any{"active": true}}); err != nil {
			t.Fatal(err)
		}
		if ok, err := s.GrantStampWeek(ctx, uid, now); err != nil || ok {
			t.Fatalf("grant: %v %v", ok, err)
		}
	})
}

func TestTrialReminderQueue(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewEntitlements(c)
	now := time.Now().Truncate(time.Second)
	put := func(doc map[string]any) string {
		uid := model.NewID()
		if _, err := c.Collection(entitlementsCollection).Doc(uid).Set(ctx, doc); err != nil {
			t.Fatal(err)
		}
		return uid
	}
	soon := now.Add(24 * time.Hour)
	due := put(map[string]any{"trial": map[string]any{"endsAt": soon}})
	skipped := map[string]bool{
		put(map[string]any{"trial": map[string]any{"endsAt": now.Add(10 * 24 * time.Hour)}}):                 true,
		put(map[string]any{"trial": map[string]any{"endsAt": soon}, "plus": map[string]any{"active": true}}): true,
		put(map[string]any{"trial": map[string]any{"endsAt": soon, "reminderFor": soon}}):                    true,
	}

	got, err := s.EndingBetween(ctx, now, now.Add(48*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, g := range got {
		found = found || g.UID == due
		if skipped[g.UID] {
			t.Errorf("trial %v must not be listed", g)
		}
	}
	if !found {
		t.Fatalf("due trial missing from %v", got)
	}
	if ok, err := s.Claim(ctx, due, soon); err != nil || !ok {
		t.Fatalf("first claim: %v %v", ok, err)
	}
	if ok, err := s.Claim(ctx, due, soon); err != nil || ok {
		t.Fatalf("second claim: %v %v", ok, err)
	}
	// An extended trial has a new end date and is reminded again.
	later := soon.Add(StampWeek)
	if _, err := c.Collection(entitlementsCollection).Doc(due).Set(ctx, map[string]any{"trial": map[string]any{"endsAt": later}}, firestore.MergeAll); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.Claim(ctx, due, later); err != nil || !ok {
		t.Fatalf("claim after extension: %v %v", ok, err)
	}
}

func TestDevPlanMarks(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewEntitlements(c)

	if got, err := s.DevPlan(ctx, model.NewID()); err != nil || got != "" {
		t.Fatalf("ordinary account: %q, %v", got, err)
	}
	for _, tt := range []struct {
		plan     string
		wantPlus bool
	}{{"free", false}, {"plus", true}, {"pass", false}, {"free", false}} {
		uid := model.NewID()
		// A mock that was Plus before must not keep it when re-marked.
		if err := s.SetDevPlan(ctx, uid, "plus"); err != nil {
			t.Fatal(err)
		}
		if err := s.SetDevPlan(ctx, uid, tt.plan); err != nil {
			t.Fatal(err)
		}
		if got, err := s.DevPlan(ctx, uid); err != nil || got != tt.plan {
			t.Errorf("DevPlan after %q = %q, %v", tt.plan, got, err)
		}
		if plus, _, err := s.Account(ctx, uid); err != nil || plus != tt.wantPlus {
			t.Errorf("plan %q: plus = %v, %v, want %v", tt.plan, plus, err, tt.wantPlus)
		}
	}
}
