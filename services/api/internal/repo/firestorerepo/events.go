package firestorerepo

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/analytics"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const eventsCollection = "analyticsEvents"

// Events implements repo.Events on the analyticsEvents collection.
type Events struct {
	client *firestore.Client
}

var _ repo.Events = (*Events)(nil)

// NewEvents returns the Firestore-backed event store.
func NewEvents(client *firestore.Client) *Events { return &Events{client: client} }

// Add stores the events. One document per event, with the person's id so funnels
// can be counted; account deletion does not remove them yet (see docs/privacy.md).
func (s *Events) Add(ctx context.Context, uid string, events []analytics.Event, now time.Time) error {
	bw := s.client.BulkWriter(ctx)
	defer bw.End()
	coll := s.client.Collection(eventsCollection)
	for _, e := range events {
		doc := map[string]any{
			"uid": uid, "name": e.Name, "at": e.At, "day": e.At.Format("2006-01-02"),
			"props": e.Props, "receivedAt": now.UTC(),
		}
		if _, err := bw.Create(coll.NewDoc(), doc); err != nil {
			return mapErr("add event", err)
		}
	}
	bw.Flush()
	return nil
}
