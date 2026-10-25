package firestorerepo

import (
	"errors"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestComments(t *testing.T) {
	client := testClient(t)
	trips, comments := NewTrips(client), NewComments(client)
	ctx := testCtx(t)
	trip, owner := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")

	c := model.Comment{ID: model.NewID(), TargetType: model.TargetStop, TargetID: "s1", AuthorUID: owner, Body: "Nice.", Mentions: []string{"u2"}}
	got, err := comments.Create(ctx, trip.ID, c)
	if err != nil || got.CreatedAt.IsZero() || !got.EditedAt.IsZero() || len(got.Mentions) != 1 {
		t.Fatalf("Create: %+v, %v", got, err)
	}
	if _, err := comments.Create(ctx, trip.ID, c); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate: %v", err)
	}
	if _, err := comments.Create(ctx, trip.ID, model.Comment{ID: "x", AuthorUID: owner, TargetID: "s1", TargetType: "poll", Body: "x"}); !model.IsValidation(err) {
		t.Errorf("bad target: %v", err)
	}
	second := model.Comment{ID: model.NewID(), TargetType: model.TargetTrip, TargetID: trip.ID, AuthorUID: owner, Body: "Later."}
	if _, err := comments.Create(ctx, trip.ID, second); err != nil {
		t.Fatal(err)
	}

	edited, err := comments.Edit(ctx, trip.ID, c.ID, "Nice, really.", nil, time.Now())
	if err != nil || edited.Body != "Nice, really." || edited.EditedAt.IsZero() || len(edited.Mentions) != 0 || !edited.CreatedAt.Equal(got.CreatedAt) {
		t.Fatalf("Edit: %+v, %v", edited, err)
	}
	if _, err := comments.Edit(ctx, trip.ID, "missing", "x", nil, time.Now()); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("edit missing: %v", err)
	}
	if _, err := comments.Edit(ctx, trip.ID, c.ID, " ", nil, time.Now()); !model.IsValidation(err) {
		t.Errorf("empty body: %v", err)
	}
	list, err := comments.List(ctx, trip.ID)
	if err != nil || len(list) != 2 || list[0].ID != c.ID {
		t.Fatalf("List: %+v, %v", list, err)
	}
	if err := comments.Delete(ctx, trip.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	if err := comments.Delete(ctx, trip.ID, c.ID); err != nil {
		t.Errorf("delete twice: %v", err)
	}
}
