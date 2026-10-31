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
