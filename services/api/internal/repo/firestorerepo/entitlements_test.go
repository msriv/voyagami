package firestorerepo

import (
	"testing"
	"time"

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
			return put(t, map[string]any{"usage": map[string]any{"askDrafts": 2}})
		}, false, 2},
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
