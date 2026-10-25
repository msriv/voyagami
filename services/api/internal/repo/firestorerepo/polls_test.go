package firestorerepo

import (
	"errors"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestPolls(t *testing.T) {
	client := testClient(t)
	trips, polls := NewTrips(client), NewPolls(client)
	ctx := testCtx(t)
	trip, owner := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")
	now := time.Now()

	p := model.Poll{ID: model.NewID(), Question: "Dinner?", CreatedBy: owner, Options: []model.PollOption{
		{ID: "a", Label: "Pho", Place: &model.PlaceRef{Name: "Pho 10", Lat: 21.03, Lng: 105.85}},
		{ID: "b", Label: "Bun cha"},
	}}
	got, err := polls.Create(ctx, trip.ID, p)
	if err != nil || got.Status != model.PollOpen || got.Options[0].Place == nil || got.Options[0].Place.Name != "Pho 10" || got.Options[1].Place != nil {
		t.Fatalf("Create: %+v, %v", got, err)
	}
	if _, err := polls.Create(ctx, trip.ID, p); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate: %v", err)
	}
	if _, err := polls.Create(ctx, trip.ID, model.Poll{ID: "x", CreatedBy: owner}); !model.IsValidation(err) {
		t.Errorf("invalid: %v", err)
	}

	if err := polls.Vote(ctx, trip.ID, p.ID, model.Vote{UID: "u1", OptionIDs: []string{"nope"}}, now); !model.IsValidation(err) {
		t.Errorf("foreign option: %v", err)
	}
	if err := polls.Vote(ctx, trip.ID, "missing", model.Vote{UID: "u1", OptionIDs: []string{"a"}}, now); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("missing poll: %v", err)
	}
	for _, v := range []model.Vote{{UID: "u1", OptionIDs: []string{"a"}}, {UID: "u2", OptionIDs: []string{"b"}}, {UID: "u2", OptionIDs: []string{"a"}}} {
		if err := polls.Vote(ctx, trip.ID, p.ID, v, now); err != nil {
			t.Fatal(err)
		}
	}
	votes, err := polls.Votes(ctx, trip.ID, p.ID)
	if err != nil || len(votes) != 2 || votes[1].UID != "u2" || votes[1].OptionIDs[0] != "a" {
		t.Fatalf("Votes: %+v, %v", votes, err)
	}

	closed, err := polls.Close(ctx, trip.ID, p.ID, "", now)
	if err != nil || closed.Status != model.PollClosed || closed.Result == nil || closed.Result.OptionID != "a" || closed.Result.Counts["a"] != 2 || closed.Result.DecidedBy != "vote" {
		t.Fatalf("Close: %+v, %v", closed, err)
	}
	again, err := polls.Close(ctx, trip.ID, p.ID, "b", now)
	if err != nil || again.Result.OptionID != "a" {
		t.Errorf("close twice: %+v, %v", again, err)
	}
	if err := polls.Vote(ctx, trip.ID, p.ID, model.Vote{UID: "u3", OptionIDs: []string{"a"}}, now); !errors.Is(err, repo.ErrConflict) {
		t.Errorf("vote after close: %v", err)
	}

	// A tie needs a pick; the pick is recorded as the owner's.
	tie := model.Poll{ID: model.NewID(), Question: "Where?", CreatedBy: owner, Options: []model.PollOption{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}}
	if _, err := polls.Create(ctx, trip.ID, tie); err != nil {
		t.Fatal(err)
	}
	if _, err := polls.Close(ctx, trip.ID, tie.ID, "", now); !errors.Is(err, model.ErrPollNoWinner) {
		t.Errorf("no votes: %v", err)
	}
	picked, err := polls.Close(ctx, trip.ID, tie.ID, "b", now)
	if err != nil || picked.Result.OptionID != "b" || picked.Result.DecidedBy != model.DecidedOwner {
		t.Errorf("owner pick: %+v, %v", picked, err)
	}

	// A poll past its time takes no votes even though nobody closed it.
	late := model.Poll{ID: model.NewID(), Question: "Late?", CreatedBy: owner, ClosesAt: now.Add(-time.Hour), Options: tie.Options}
	if _, err := polls.Create(ctx, trip.ID, late); err != nil {
		t.Fatal(err)
	}
	if err := polls.Vote(ctx, trip.ID, late.ID, model.Vote{UID: "u1", OptionIDs: []string{"a"}}, now); !errors.Is(err, repo.ErrConflict) {
		t.Errorf("vote after closesAt: %v", err)
	}
	list, err := polls.List(ctx, trip.ID)
	if err != nil || len(list) != 3 {
		t.Fatalf("List: %+v, %v", list, err)
	}
}
