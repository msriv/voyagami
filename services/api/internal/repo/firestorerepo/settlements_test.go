package firestorerepo

import (
	"errors"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestSettlements(t *testing.T) {
	client := testClient(t)
	trips, pays := NewTrips(client), NewSettlements(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")

	p := model.Settlement{ID: "p1", FromUID: "b", ToUID: "a", MarkedBy: "b", Method: "manual", Amount: model.Money{AmountMinor: 333, Currency: "VND"}}
	got, err := pays.Create(ctx, trip.ID, p)
	if err != nil || got.Amount.AmountMinor != 333 || got.Method != "manual" || got.CreatedAt.IsZero() {
		t.Fatalf("Create: %+v, %v", got, err)
	}
	if _, err := pays.Create(ctx, trip.ID, p); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate: %v", err)
	}
	p.ID, p.ToUID = "p2", "b"
	if _, err := pays.Create(ctx, trip.ID, p); !model.IsValidation(err) {
		t.Errorf("paying yourself: %v", err)
	}
	if _, err := pays.Get(ctx, trip.ID, "nope"); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Get missing: %v", err)
	}
	if list, err := pays.List(ctx, trip.ID); err != nil || len(list) != 1 {
		t.Errorf("List: %+v, %v", list, err)
	}
}
