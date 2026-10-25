package firestorerepo

import (
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestJournal(t *testing.T) {
	client := testClient(t)
	trips, journal := NewTrips(client), NewJournal(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")

	e := model.JournalEntry{Date: "2026-11-22", AuthorUID: "u1", Visibility: "crew", Text: "Day one.", Mood: "happy", StopIDs: []string{"s1"}}
	first, err := journal.Put(ctx, trip.ID, e)
	if err != nil || first.CreatedAt.IsZero() {
		t.Fatalf("Put: %+v, %v", first, err)
	}
	e.Text, e.Mood, e.Visibility = "Day one, edited.", "", "private"
	second, err := journal.Put(ctx, trip.ID, e)
	if err != nil || second.Text != "Day one, edited." || !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("second Put: %+v, %v", second, err)
	}
	if _, err := journal.Put(ctx, trip.ID, model.JournalEntry{Date: "nope", AuthorUID: "u1", Visibility: "crew"}); !model.IsValidation(err) {
		t.Errorf("bad date: %v", err)
	}
	other := model.JournalEntry{Date: "2026-11-22", AuthorUID: "u0", Visibility: "crew", Text: "Hi."}
	later := model.JournalEntry{Date: "2026-11-23", AuthorUID: "u1", Visibility: "crew", Text: "Day two."}
	for _, x := range []model.JournalEntry{other, later} {
		if _, err := journal.Put(ctx, trip.ID, x); err != nil {
			t.Fatal(err)
		}
	}
	list, err := journal.List(ctx, trip.ID)
	if err != nil || len(list) != 3 || list[0].Date != "2026-11-23" || list[1].AuthorUID != "u0" || list[2].Visibility != "private" || list[2].Mood != "" {
		t.Fatalf("List: %+v, %v", list, err)
	}
	if err := journal.Delete(ctx, trip.ID, "2026-11-22", "u1"); err != nil {
		t.Fatal(err)
	}
	if err := journal.Delete(ctx, trip.ID, "2026-11-22", "u1"); err != nil {
		t.Errorf("delete twice: %v", err)
	}
	if list, _ := journal.List(ctx, trip.ID); len(list) != 2 {
		t.Errorf("after delete: %+v", list)
	}
}
