package firestorerepo

import (
	"errors"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestExpensesLifecycle(t *testing.T) {
	client := testClient(t)
	trips, expenses := NewTrips(client), NewExpenses(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")

	mk := func(date string) model.Expense {
		return model.Expense{
			ID: model.NewID(), Description: "Pho", PaidByUID: "u1", Date: date, CreatedBy: "u1",
			Original:    model.Money{AmountMinor: 250000, Currency: "VND"},
			Settlement:  model.Money{AmountMinor: 250000, Currency: "VND"},
			SplitMethod: "equal",
			Shares:      []model.Share{{UID: "u1", AmountMinor: 125000}, {UID: "u2", AmountMinor: 125000}},
		}
	}
	older, err := expenses.Create(ctx, trip.ID, mk("2026-11-22"))
	if err != nil || older.CreatedAt.IsZero() || len(older.Shares) != 2 || older.Rate != nil {
		t.Fatalf("Create: %+v, %v", older, err)
	}
	foreign := mk("2026-11-23")
	foreign.Original = model.Money{AmountMinor: 1000, Currency: "USD"}
	foreign.Rate = &model.Rate{Numerator: 250, Denominator: 1}
	newer, err := expenses.Create(ctx, trip.ID, foreign)
	if err != nil || newer.Rate == nil || newer.Rate.Numerator != 250 || newer.Original.Currency != "USD" {
		t.Fatalf("Create foreign: %+v, %v", newer, err)
	}
	if _, err := expenses.Create(ctx, trip.ID, older); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate: %v", err)
	}
	bad := mk("2026-11-24")
	bad.Shares[0].AmountMinor = 1
	if _, err := expenses.Create(ctx, trip.ID, bad); !model.IsValidation(err) {
		t.Errorf("shares that do not add up: %v", err)
	}

	list, err := expenses.List(ctx, trip.ID)
	if err != nil || len(list) != 2 || list[0].ID != newer.ID {
		t.Fatalf("List: %+v, %v", list, err)
	}

	edit := older
	edit.Description, edit.CreatedBy = "Pho and tea", "someone-else"
	got, err := expenses.Replace(ctx, trip.ID, edit)
	if err != nil || got.Description != "Pho and tea" || got.CreatedBy != "u1" || !got.CreatedAt.Equal(older.CreatedAt) {
		t.Fatalf("Replace: %+v, %v", got, err)
	}

	if err := expenses.Delete(ctx, trip.ID, older.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := expenses.Get(ctx, trip.ID, older.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Get deleted: %v", err)
	}
	if _, err := expenses.Replace(ctx, trip.ID, edit); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Replace deleted: %v", err)
	}
	if err := expenses.Delete(ctx, trip.ID, "missing"); err != nil {
		t.Errorf("Delete missing: %v", err)
	}
	if list, _ := expenses.List(ctx, trip.ID); len(list) != 1 {
		t.Errorf("list after delete: %+v", list)
	}
}
