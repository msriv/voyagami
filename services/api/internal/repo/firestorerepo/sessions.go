package firestorerepo

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Sessions implements repo.Sessions.
type Sessions struct {
	client *firestore.Client
	now    func() time.Time
}

var _ repo.Sessions = (*Sessions)(nil)

// NewSessions returns the Firestore-backed session store.
func NewSessions(client *firestore.Client) *Sessions {
	return &Sessions{client: client, now: time.Now}
}

type sessionDoc struct {
	UserID      string     `firestore:"userId"`
	RefreshHash string     `firestore:"refreshHash"`
	ExpiresAt   time.Time  `firestore:"expiresAt"`
	RevokedAt   *time.Time `firestore:"revokedAt,omitempty"`
	CreatedAt   time.Time  `firestore:"createdAt,serverTimestamp"`
	UpdatedAt   time.Time  `firestore:"updatedAt,serverTimestamp"`
}

func (s *Sessions) ref(id string) *firestore.DocumentRef {
	return s.client.Collection(sessionsCollection).Doc(id)
}

// Create stores a new session.
func (s *Sessions) Create(ctx context.Context, m model.Session) error {
	if m.ID == "" || m.UserID == "" || m.RefreshHash == "" {
		return repo.ErrNotFound
	}
	_, err := s.ref(m.ID).Create(ctx, sessionDoc{UserID: m.UserID, RefreshHash: m.RefreshHash, ExpiresAt: m.ExpiresAt})
	return mapErr("create session", err)
}

// Get returns one session.
func (s *Sessions) Get(ctx context.Context, id string) (model.Session, error) {
	if id == "" {
		return model.Session{}, repo.ErrNotFound
	}
	snap, err := s.ref(id).Get(ctx)
	if err != nil {
		return model.Session{}, mapErr("get session", err)
	}
	var d sessionDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Session{}, mapErr("decode session", err)
	}
	return model.Session{
		ID: id, UserID: d.UserID, RefreshHash: d.RefreshHash, ExpiresAt: d.ExpiresAt,
		RevokedAt: d.RevokedAt, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}, nil
}

// Rotate swaps the refresh hash, or revokes the session if the old hash is stale.
func (s *Sessions) Rotate(ctx context.Context, id, oldHash, newHash string, expiresAt time.Time) error {
	if id == "" {
		return repo.ErrNotFound
	}
	ref := s.ref(id)
	reused := false
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		reused = false
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var d sessionDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		if d.RevokedAt != nil || !s.now().Before(d.ExpiresAt) {
			return repo.ErrNotFound
		}
		now := s.now().UTC()
		if d.RefreshHash != oldHash {
			reused = true
			return tx.Update(ref, []firestore.Update{
				{Path: "revokedAt", Value: now},
				{Path: "updatedAt", Value: firestore.ServerTimestamp},
			})
		}
		return tx.Update(ref, []firestore.Update{
			{Path: "refreshHash", Value: newHash},
			{Path: "expiresAt", Value: expiresAt},
			{Path: "updatedAt", Value: firestore.ServerTimestamp},
		})
	})
	if err != nil {
		return mapErr("rotate session", err)
	}
	if reused {
		return repo.ErrTokenReuse
	}
	return nil
}

// Revoke ends a session. It is safe to call twice.
func (s *Sessions) Revoke(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	_, err := s.ref(id).Update(ctx, []firestore.Update{
		{Path: "revokedAt", Value: s.now().UTC()},
		{Path: "updatedAt", Value: firestore.ServerTimestamp},
	})
	if isNotFound(err) {
		return nil
	}
	return mapErr("revoke session", err)
}
