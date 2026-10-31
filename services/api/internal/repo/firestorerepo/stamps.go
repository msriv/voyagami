package firestorerepo

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const (
	firstStampsCollection = "firstStamps"
	progressCollection    = "progress"
	placesProgressDoc     = "places"
)

// Stamps implements repo.Stamps. A stamp is the document users/{uid}/firstStamps/{kind};
// the place count is users/{uid}/progress/places.
type Stamps struct {
	client *firestore.Client
}

var _ repo.Stamps = (*Stamps)(nil)

// NewStamps returns the Firestore-backed stamp store.
func NewStamps(client *firestore.Client) *Stamps { return &Stamps{client: client} }

type stampDoc struct {
	EarnedAt time.Time `firestore:"earnedAt"`
}

func (s *Stamps) stamps(uid string) *firestore.CollectionRef {
	return s.client.Collection(usersCollection).Doc(uid).Collection(firstStampsCollection)
}

// Award creates the stamp if it is not there yet.
func (s *Stamps) Award(ctx context.Context, uid string, kind model.StampKind, at time.Time) (bool, error) {
	_, err := s.stamps(uid).Doc(string(kind)).Create(ctx, stampDoc{at.UTC().Truncate(time.Microsecond)})
	if isAlreadyExists(err) {
		return false, nil
	}
	return err == nil, mapErr("award stamp", err)
}

// AddPlace counts a place and awards the stamp when the count reaches the target.
func (s *Stamps) AddPlace(ctx context.Context, uid string, at time.Time) error {
	ref := s.client.Collection(usersCollection).Doc(uid).Collection(progressCollection).Doc(placesProgressDoc)
	stamp := s.stamps(uid).Doc(string(model.StampAddPlaces))
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		count := 0
		snap, err := tx.Get(ref)
		switch {
		case isNotFound(err):
		case err != nil:
			return err
		default:
			n, _ := snap.DataAt("count")
			if v, ok := n.(int64); ok {
				count = int(v)
			}
		}
		count++
		if err := tx.Set(ref, map[string]any{"count": count}); err != nil {
			return err
		}
		if count == model.PlacesForStamp {
			return tx.Set(stamp, stampDoc{at.UTC().Truncate(time.Microsecond)})
		}
		return nil
	})
	return mapErr("count place", err)
}

// List returns the stamps earned.
func (s *Stamps) List(ctx context.Context, uid string) ([]model.EarnedStamp, error) {
	iter := s.stamps(uid).Documents(ctx)
	defer iter.Stop()
	var out []model.EarnedStamp
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, mapErr("list stamps", err)
		}
		var d stampDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode stamp", err)
		}
		out = append(out, model.EarnedStamp{Kind: model.StampKind(snap.Ref.ID), EarnedAt: d.EarnedAt})
	}
}
