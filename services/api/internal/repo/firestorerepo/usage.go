package firestorerepo

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Usage implements repo.TripUsage on the trip document's usage map.
type Usage struct {
	client *firestore.Client
}

var _ repo.TripUsage = (*Usage)(nil)

// NewUsage returns the Firestore-backed usage counters.
func NewUsage(client *firestore.Client) *Usage { return &Usage{client: client} }

// Add raises one counter by n.
func (s *Usage) Add(ctx context.Context, tripID string, c model.UsageCounter, n int) error {
	if tripID == "" {
		return repo.ErrNotFound
	}
	_, err := s.client.Collection(tripsCollection).Doc(tripID).Update(ctx, []firestore.Update{
		{Path: "usage." + string(c), Value: firestore.Increment(n)},
	})
	return mapErr("add usage", err)
}
