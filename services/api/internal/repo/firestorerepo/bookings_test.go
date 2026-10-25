package firestorerepo

import (
	"errors"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestBookings(t *testing.T) {
	client := testClient(t)
	trips, bookings := NewTrips(client), NewBookings(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")

	day := time.Date(2026, 11, 21, 3, 0, 0, 0, time.UTC)
	flight := model.Booking{
		ID: "b1", Kind: "flight", Title: "BLR to HAN", Provider: "VietJet", StartsAt: day, EndsAt: day.Add(5 * time.Hour),
		StartTimeZone: "Asia/Kolkata", From: &model.BookingPlace{Code: "BLR", Name: "Bengaluru"},
		Price: &model.Money{AmountMinor: 1250000, Currency: "INR"}, CreatedBy: "u1",
	}
	got, err := bookings.Create(ctx, trip.ID, flight)
	if err != nil || !got.StartsAt.Equal(day) || got.From.Code != "BLR" || got.Price.AmountMinor != 1250000 || got.CreatedAt.IsZero() {
		t.Fatalf("Create: %+v, %v", got, err)
	}
	if _, err := bookings.Create(ctx, trip.ID, flight); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate: %v", err)
	}
	if _, err := bookings.Create(ctx, trip.ID, model.Booking{ID: "b2", Kind: "boat", Title: "x"}); !model.IsValidation(err) {
		t.Errorf("bad kind: %v", err)
	}
	undated, _ := bookings.Create(ctx, trip.ID, model.Booking{ID: "b0", Kind: "stay", Title: "Hotel", CreatedBy: "u1"})
	list, err := bookings.List(ctx, trip.ID)
	if err != nil || len(list) != 2 || list[0].ID != "b1" || list[1].ID != undated.ID {
		t.Fatalf("List: %+v, %v", list, err)
	}

	edit := got
	edit.Provider, edit.From, edit.Price, edit.CreatedBy = "", nil, nil, "someone-else"
	edit.Title = "Direct"
	got2, err := bookings.Replace(ctx, trip.ID, edit)
	if err != nil || got2.Title != "Direct" || got2.Provider != "" || got2.From != nil || got2.Price != nil || got2.CreatedBy != "u1" {
		t.Fatalf("Replace: %+v, %v", got2, err)
	}
	if _, err := bookings.Replace(ctx, trip.ID, model.Booking{ID: "nope", Kind: "stay", Title: "x"}); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("Replace missing: %v", err)
	}
	if err := bookings.Delete(ctx, trip.ID, "b1"); err != nil {
		t.Fatal(err)
	}
	if err := bookings.Delete(ctx, trip.ID, "b1"); err != nil {
		t.Errorf("Delete twice: %v", err)
	}
}
