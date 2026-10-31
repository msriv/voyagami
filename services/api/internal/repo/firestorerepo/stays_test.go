package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestStayClicksCountPerDay(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewStayClicks(c)
	partner := "p-" + model.NewID()
	day1 := time.Date(2026, 12, 1, 23, 59, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if err := s.Add(ctx, partner, day1); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Add(ctx, partner, day1.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Count(ctx, partner, day1); n != 3 {
		t.Errorf("day one = %d, want 3", n)
	}
	if n, _ := s.Count(ctx, partner, day1.Add(2*time.Minute)); n != 1 {
		t.Errorf("day two = %d, want 1", n)
	}
}
