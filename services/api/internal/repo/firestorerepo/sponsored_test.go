package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestSponsored(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewSponsored(c)
	now := time.Now()
	_, _ = c.Collection(sponsoredCollection).Doc("on").Set(ctx, sponsoredDoc{Active: true, Sponsor: "S", Title: "T", URL: "https://x.example",
		Countries: []string{"VN"}, Groups: []string{"eat"}, Priority: 2, StartsAt: now.Add(-time.Hour)})
	_, _ = c.Collection(sponsoredCollection).Doc("off").Set(ctx, sponsoredDoc{Active: false, StartsAt: now.Add(-time.Hour)})
	_, _ = c.Collection(sponsoredCollection).Doc("later").Set(ctx, sponsoredDoc{Active: true, StartsAt: now.Add(time.Hour)})

	live, err := s.Live(ctx, now)
	if err != nil || len(live) != 1 || live[0].ID != "on" || live[0].Countries[0] != "VN" || live[0].Groups[0] != "eat" {
		t.Fatalf("live: %+v %v", live, err)
	}
	if _, err := s.Get(ctx, "missing"); !isNotFound(err) && err == nil {
		t.Errorf("missing card: %v", err)
	}

	if err := s.Hide(ctx, "u1", "on", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if h, _ := s.Hidden(ctx, "u1", now); !h["on"] {
		t.Error("should be hidden")
	}
	if h, _ := s.Hidden(ctx, "u1", now.Add(2*time.Hour)); h["on"] {
		t.Error("hide should expire")
	}

	for i := 0; i < 3; i++ {
		if err := s.Record(ctx, "u1", "on", model.SponsoredImpression, now); err != nil {
			t.Fatal(err)
		}
	}
	_ = s.Record(ctx, "u2", "on", model.SponsoredImpression, now)
	snap, err := c.Collection(sponsoredStatsCollection).Doc("on_" + now.UTC().Format("2006-01-02")).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := snap.DataAt("impressions"); n != int64(2) {
		t.Errorf("impressions %v, want 2 (one per person per day)", n)
	}
}
