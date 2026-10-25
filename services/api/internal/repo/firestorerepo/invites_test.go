package firestorerepo

import (
	"errors"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestInvites(t *testing.T) {
	client := testClient(t)
	trips, invites := NewTrips(client), NewInvites(client)
	ctx := testCtx(t)
	trip, _ := newOwnedTrip(t, trips, "2026-11-21", "2026-11-29")
	now := time.Now().UTC().Truncate(time.Microsecond)

	inv := model.Invite{ID: "hash-" + trip.ID, TripID: trip.ID, Role: model.RoleEditor, CreatedBy: "u1", CreatedAt: now, ExpiresAt: now.Add(time.Hour), MaxUses: 2}
	if err := invites.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	if err := invites.Create(ctx, inv); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate: %v", err)
	}
	if err := invites.Create(ctx, model.Invite{ID: "x", TripID: trip.ID, Role: model.RoleOwner, CreatedBy: "u1"}); !model.IsValidation(err) {
		t.Errorf("owner role: %v", err)
	}
	for i := 1; i <= 2; i++ {
		if got, err := invites.Use(ctx, inv.ID, now); err != nil || got.UseCount != i || got.Role != model.RoleEditor {
			t.Fatalf("use %d: %+v, %v", i, got, err)
		}
	}
	if _, err := invites.Use(ctx, inv.ID, now); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("used up: %v", err)
	}
	if _, err := invites.Use(ctx, "missing", now); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("missing: %v", err)
	}

	open := model.Invite{ID: "open-" + trip.ID, TripID: trip.ID, Role: model.RoleViewer, CreatedBy: "u1", CreatedAt: now.Add(time.Second)}
	if err := invites.Create(ctx, open); err != nil {
		t.Fatal(err)
	}
	if _, err := invites.Use(ctx, inv.ID, now.Add(2*time.Hour)); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("expired: %v", err)
	}
	if err := invites.Revoke(ctx, "other-trip", open.ID, now); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("revoke on another trip: %v", err)
	}
	if err := invites.Revoke(ctx, trip.ID, open.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := invites.Revoke(ctx, trip.ID, open.ID, now); err != nil {
		t.Errorf("revoke twice: %v", err)
	}
	if _, err := invites.Use(ctx, open.ID, now); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("revoked: %v", err)
	}
	list, err := invites.List(ctx, trip.ID)
	if err != nil || len(list) != 2 || list[0].ID != open.ID || list[0].RevokedAt.IsZero() || list[1].UseCount != 2 {
		t.Fatalf("List: %+v, %v", list, err)
	}
}
