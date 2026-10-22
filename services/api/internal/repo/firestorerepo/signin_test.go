package firestorerepo

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func newSignInUser() model.User {
	return model.User{
		ID: uuid.NewString(), HomeRegion: "ZZ", HomeCurrency: "XXX", Language: "en", RegionSource: "device",
		Units: model.Units{Distance: "km", Temperature: "c", Clock: "24h", FirstWeekday: 1},
	}
}

func TestAccountsFindOrCreate(t *testing.T) {
	ctx := testCtx(t)
	accounts := NewAccounts(testClient(t))
	hash := uuid.NewString()

	first := newSignInUser()
	u, created, err := accounts.FindOrCreate(ctx, hash, first)
	if err != nil || !created || u.ID != first.ID {
		t.Fatalf("first = %+v, %v, %v", u, created, err)
	}

	// A second sign-in with the same hash finds the first user and creates nothing.
	other := newSignInUser()
	u, created, err = accounts.FindOrCreate(ctx, hash, other)
	if err != nil || created || u.ID != first.ID {
		t.Fatalf("second = %+v, %v, %v; want the first user", u, created, err)
	}
	if _, err := NewUsers(accounts.client).Get(ctx, other.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("second candidate user was stored: %v", err)
	}

	if _, _, err := accounts.FindOrCreate(ctx, hash, model.User{ID: "x"}); err == nil {
		t.Error("invalid user accepted")
	}
}

func TestSessions(t *testing.T) {
	ctx := testCtx(t)
	s := NewSessions(testClient(t))
	exp := time.Now().Add(time.Hour)
	sess := model.Session{ID: uuid.NewString(), UserID: "u1", RefreshHash: "h1", ExpiresAt: exp}

	if err := s.Create(ctx, sess); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(ctx, sess); !errors.Is(err, repo.ErrAlreadyExists) {
		t.Errorf("duplicate create = %v, want ErrAlreadyExists", err)
	}
	got, err := s.Get(ctx, sess.ID)
	if err != nil || got.UserID != "u1" || got.RefreshHash != "h1" || got.RevokedAt != nil {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if _, err := s.Get(ctx, "missing"); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("get missing = %v, want ErrNotFound", err)
	}

	if err := s.Rotate(ctx, sess.ID, "h1", "h2", exp.Add(time.Hour)); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if got, _ := s.Get(ctx, sess.ID); got.RefreshHash != "h2" {
		t.Errorf("hash = %q, want h2", got.RefreshHash)
	}

	// The old hash again means a stolen or replayed token: the session ends.
	if err := s.Rotate(ctx, sess.ID, "h1", "h3", exp); !errors.Is(err, repo.ErrTokenReuse) {
		t.Fatalf("replay = %v, want ErrTokenReuse", err)
	}
	if got, _ := s.Get(ctx, sess.ID); got.RevokedAt == nil {
		t.Error("session not revoked after reuse")
	}
	if err := s.Rotate(ctx, sess.ID, "h2", "h3", exp); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("rotate revoked = %v, want ErrNotFound", err)
	}
}

func TestSessionsRevokeAndExpiry(t *testing.T) {
	ctx := testCtx(t)
	s := NewSessions(testClient(t))

	live := model.Session{ID: uuid.NewString(), UserID: "u1", RefreshHash: "a", ExpiresAt: time.Now().Add(time.Hour)}
	if err := s.Create(ctx, live); err != nil {
		t.Fatal(err)
	}
	if err := s.Revoke(ctx, live.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Revoke(ctx, live.ID); err != nil {
		t.Errorf("second revoke = %v", err)
	}
	if err := s.Revoke(ctx, "missing"); err != nil {
		t.Errorf("revoke missing = %v", err)
	}
	if err := s.Rotate(ctx, live.ID, "a", "b", time.Now().Add(time.Hour)); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("rotate after revoke = %v, want ErrNotFound", err)
	}

	old := model.Session{ID: uuid.NewString(), UserID: "u1", RefreshHash: "a", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := s.Create(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := s.Rotate(ctx, old.ID, "a", "b", time.Now().Add(time.Hour)); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("rotate expired = %v, want ErrNotFound", err)
	}
}
