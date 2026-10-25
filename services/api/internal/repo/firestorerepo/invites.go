package firestorerepo

import (
	"context"
	"errors"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const invitesCollection = "invites"

// Invites implements repo.Invites.
type Invites struct {
	client *firestore.Client
}

var _ repo.Invites = (*Invites)(nil)

// NewInvites returns the Firestore-backed invite store.
func NewInvites(client *firestore.Client) *Invites { return &Invites{client: client} }

type inviteDoc struct {
	TripID    string    `firestore:"tripId"`
	Role      string    `firestore:"role"`
	CreatedBy string    `firestore:"createdBy"`
	CreatedAt time.Time `firestore:"createdAt"`
	ExpiresAt time.Time `firestore:"expiresAt,omitempty"`
	MaxUses   int       `firestore:"maxUses"`
	UseCount  int       `firestore:"useCount"`
	RevokedAt time.Time `firestore:"revokedAt,omitempty"`
}

func (s *Invites) ref(id string) *firestore.DocumentRef {
	return s.client.Collection(invitesCollection).Doc(id)
}

func inviteFromDoc(id string, d inviteDoc) model.Invite {
	return model.Invite{
		ID: id, TripID: d.TripID, Role: model.Role(d.Role), CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt,
		ExpiresAt: d.ExpiresAt, MaxUses: d.MaxUses, UseCount: d.UseCount, RevokedAt: d.RevokedAt,
	}
}

// Create stores the invite.
func (s *Invites) Create(ctx context.Context, i model.Invite) error {
	if err := i.Validate(); err != nil {
		return err
	}
	_, err := s.ref(i.ID).Create(ctx, inviteDoc{
		TripID: i.TripID, Role: string(i.Role), CreatedBy: i.CreatedBy, CreatedAt: i.CreatedAt.UTC().Truncate(time.Microsecond),
		ExpiresAt: i.ExpiresAt.UTC().Truncate(time.Microsecond), MaxUses: i.MaxUses,
	})
	return mapErr("create invite", err)
}

// Get returns one invite.
func (s *Invites) Get(ctx context.Context, id string) (model.Invite, error) {
	if id == "" {
		return model.Invite{}, repo.ErrNotFound
	}
	snap, err := s.ref(id).Get(ctx)
	if err != nil {
		return model.Invite{}, mapErr("get invite", err)
	}
	var d inviteDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Invite{}, mapErr("decode invite", err)
	}
	return inviteFromDoc(id, d), nil
}

// List returns the trip's invites, newest first.
func (s *Invites) List(ctx context.Context, tripID string) ([]model.Invite, error) {
	iter := s.client.Collection(invitesCollection).Where("tripId", "==", tripID).Documents(ctx)
	defer iter.Stop()
	var out []model.Invite
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list invites", err)
		}
		var d inviteDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode invite", err)
		}
		out = append(out, inviteFromDoc(snap.Ref.ID, d))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Revoke ends the invite.
func (s *Invites) Revoke(ctx context.Context, tripID, id string, now time.Time) error {
	ref := s.ref(id)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var d inviteDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		if d.TripID != tripID {
			return repo.ErrNotFound
		}
		if !d.RevokedAt.IsZero() {
			return nil
		}
		return tx.Update(ref, []firestore.Update{{Path: "revokedAt", Value: now.UTC().Truncate(time.Microsecond)}})
	})
	return mapErr("revoke invite", err)
}

// Use counts one join if the invite is still usable.
func (s *Invites) Use(ctx context.Context, id string, now time.Time) (model.Invite, error) {
	if id == "" {
		return model.Invite{}, repo.ErrNotFound
	}
	ref := s.ref(id)
	var out model.Invite
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var d inviteDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		inv := inviteFromDoc(id, d)
		if !inv.Usable(now) {
			return repo.ErrNotFound
		}
		inv.UseCount++
		out = inv
		return tx.Update(ref, []firestore.Update{{Path: "useCount", Value: inv.UseCount}})
	})
	if err != nil {
		return model.Invite{}, mapErr("use invite", err)
	}
	return out, nil
}
