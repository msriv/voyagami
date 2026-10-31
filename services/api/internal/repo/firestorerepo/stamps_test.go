package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestStamps(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewStamps(c)
	uid := model.NewID()
	at := time.Now()

	if won, err := s.Award(ctx, uid, model.StampCreateTrip, at); err != nil || !won {
		t.Fatalf("first award: won=%v err=%v", won, err)
	}
	if won, err := s.Award(ctx, uid, model.StampCreateTrip, at.Add(time.Hour)); err != nil || won {
		t.Fatalf("second award: won=%v err=%v", won, err)
	}
	for i := 0; i < model.PlacesForStamp-1; i++ {
		if won, err := s.AddPlace(ctx, uid, at); err != nil || won {
			t.Fatalf("early place: won=%v err=%v", won, err)
		}
	}
	got, err := s.List(ctx, uid)
	if err != nil || len(got) != 1 {
		t.Fatalf("after two places: %v %v", got, err)
	}
	if won, err := s.AddPlace(ctx, uid, at); err != nil || !won {
		t.Fatalf("third place: won=%v err=%v", won, err)
	}
	if got, _ = s.List(ctx, uid); len(got) != 2 {
		t.Fatalf("third place must earn the stamp: %v", got)
	}
	// Later places change nothing.
	if won, err := s.AddPlace(ctx, uid, at.Add(time.Hour)); err != nil || won {
		t.Fatalf("fourth place: won=%v err=%v", won, err)
	}
	if got, _ = s.List(ctx, uid); len(got) != 2 {
		t.Fatalf("fourth place: %v", got)
	}
}
