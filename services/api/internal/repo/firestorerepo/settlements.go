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

const settlementsCollection = "settlements"

// Settlements implements repo.Settlements.
type Settlements struct {
	client *firestore.Client
}

var _ repo.Settlements = (*Settlements)(nil)

// NewSettlements returns the Firestore-backed payment store.
func NewSettlements(client *firestore.Client) *Settlements { return &Settlements{client: client} }

type settlementDoc struct {
	FromUID   string    `firestore:"fromUid"`
	ToUID     string    `firestore:"toUid"`
	Amount    moneyDoc  `firestore:"amount"`
	Status    string    `firestore:"status"`
	Method    string    `firestore:"method,omitempty"`
	MarkedBy  string    `firestore:"markedBy"`
	CreatedAt time.Time `firestore:"createdAt,serverTimestamp"`
}

func (s *Settlements) col(tripID string) *firestore.CollectionRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(settlementsCollection)
}

func settlementFromSnap(snap *firestore.DocumentSnapshot) (model.Settlement, error) {
	var d settlementDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Settlement{}, mapErr("decode settlement", err)
	}
	return model.Settlement{
		ID: snap.Ref.ID, FromUID: d.FromUID, ToUID: d.ToUID, Method: d.Method, MarkedBy: d.MarkedBy, CreatedAt: d.CreatedAt,
		Amount: model.Money{AmountMinor: d.Amount.AmountMinor, Currency: d.Amount.Currency},
	}, nil
}

// Create stores a payment as paid. The ID is the idempotency key.
func (s *Settlements) Create(ctx context.Context, tripID string, p model.Settlement) (model.Settlement, error) {
	if err := p.Validate(); err != nil {
		return model.Settlement{}, err
	}
	d := settlementDoc{
		FromUID: p.FromUID, ToUID: p.ToUID, Status: "paid", Method: p.Method, MarkedBy: p.MarkedBy,
		Amount: moneyDoc{AmountMinor: p.Amount.AmountMinor, Currency: p.Amount.Currency},
	}
	if _, err := s.col(tripID).Doc(p.ID).Create(ctx, d); err != nil {
		return model.Settlement{}, mapErr("create settlement", err)
	}
	return s.Get(ctx, tripID, p.ID)
}

// Get returns one payment.
func (s *Settlements) Get(ctx context.Context, tripID, id string) (model.Settlement, error) {
	snap, err := s.col(tripID).Doc(id).Get(ctx)
	if err != nil {
		return model.Settlement{}, mapErr("get settlement", err)
	}
	return settlementFromSnap(snap)
}

// List returns the trip's payments, oldest first.
func (s *Settlements) List(ctx context.Context, tripID string) ([]model.Settlement, error) {
	iter := s.col(tripID).Documents(ctx)
	defer iter.Stop()
	var out []model.Settlement
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list settlements", err)
		}
		p, err := settlementFromSnap(snap)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
