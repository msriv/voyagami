package firestorerepo

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Accounts implements repo.Accounts.
type Accounts struct {
	client *firestore.Client
}

var _ repo.Accounts = (*Accounts)(nil)

// NewAccounts returns the Firestore-backed account index.
func NewAccounts(client *firestore.Client) *Accounts { return &Accounts{client: client} }

type phoneIndexDoc struct {
	UID       string    `firestore:"uid"`
	CreatedAt time.Time `firestore:"createdAt,serverTimestamp"`
}

// FindOrCreate looks up the user for a phone hash, creating user and link in one transaction.
func (s *Accounts) FindOrCreate(ctx context.Context, phoneHash string, newUser model.User) (model.User, bool, error) {
	if phoneHash == "" {
		return model.User{}, false, repo.ErrNotFound
	}
	if err := newUser.Validate(); err != nil {
		return model.User{}, false, err
	}
	idx := s.client.Collection(phoneIndexCollection).Doc(phoneHash)
	var uid string
	created := false
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		created = false
		snap, err := tx.Get(idx)
		if err == nil {
			var d phoneIndexDoc
			if err := snap.DataTo(&d); err != nil {
				return err
			}
			uid = d.UID
			return nil
		}
		if !isNotFound(err) {
			return err
		}
		uid, created = newUser.ID, true
		if err := tx.Create(s.client.Collection(usersCollection).Doc(newUser.ID), userToDoc(newUser)); err != nil {
			return err
		}
		return tx.Create(idx, phoneIndexDoc{UID: newUser.ID})
	})
	if err != nil {
		return model.User{}, false, mapErr("find or create account", err)
	}
	u, err := NewUsers(s.client).Get(ctx, uid)
	return u, created, err
}
