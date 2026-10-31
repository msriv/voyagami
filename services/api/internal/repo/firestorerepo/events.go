package firestorerepo

import (
	"context"
	"errors"
	"time"

	"google.golang.org/api/iterator"

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

// Rows returns the events from sinceDay (YYYY-MM-DD) on, at most limit of them.
//
// shortcut: scans the stored events, which is fine for a few thousand a week
// from dogfooding. Move to the BigQuery export of Firestore before launch traffic.
func (s *Events) Rows(ctx context.Context, sinceDay string, limit int) ([]analytics.Row, error) {
	it := s.client.Collection(eventsCollection).Where("day", ">=", sinceDay).Limit(limit).Documents(ctx)
	defer it.Stop()
	var out []analytics.Row
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, mapErr("read events", err)
		}
		m := snap.Data()
		r := analytics.Row{}
		r.UID, _ = m["uid"].(string)
		r.Name, _ = m["name"].(string)
		r.Day, _ = m["day"].(string)
		if props, ok := m["props"].(map[string]any); ok {
			r.Moment, _ = props["moment"].(string)
		}
		out = append(out, r)
	}
}
