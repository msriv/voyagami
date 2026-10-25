package firestorerepo

import (
	"context"
	"errors"
	"sync"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Feed implements repo.Feed with Firestore snapshot listeners, so it works
// across Cloud Run instances without a change log.
type Feed struct {
	client *firestore.Client
}

var _ repo.Feed = (*Feed)(nil)

// NewFeed returns the Firestore-backed change feed.
func NewFeed(client *firestore.Client) *Feed { return &Feed{client: client} }

// feedKinds are the subcollections of a trip that the feed reports. Journal
// and presence are left out: they are private to one person.
var feedKinds = []string{
	"members", stopsCollection, tasksCollection, expensesCollection, settlementsCollection,
	bookingsCollection, pollsCollection, meetPointsCollection, commentsCollection,
}

// Watch listens to the trip document and each subcollection. The first
// snapshot of each is the current state, not a change, so it is skipped.
func (f *Feed) Watch(ctx context.Context, tripID string, emit func(model.Change)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	trip := f.client.Collection(tripsCollection).Doc(tripID)

	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error
	fail := func(err error) {
		if err != nil && ctx.Err() == nil {
			once.Do(func() { firstErr = err })
		}
		cancel()
	}
	run := func(fn func() error) {
		wg.Add(1)
		go func() { defer wg.Done(); fail(fn()) }()
	}

	run(func() error {
		it := trip.Snapshots(ctx)
		defer it.Stop()
		for first := true; ; first = false {
			snap, err := it.Next()
			if err != nil {
				return err
			}
			if !first {
				op := "upsert"
				if !snap.Exists() {
					op = "delete"
				}
				emit(model.Change{Kind: "trip", ID: tripID, Op: op})
			}
		}
	})
	for _, kind := range feedKinds {
		run(func() error {
			it := trip.Collection(kind).Snapshots(ctx)
			defer it.Stop()
			for first := true; ; first = false {
				snap, err := it.Next()
				if err != nil {
					return err
				}
				if first {
					continue
				}
				for _, c := range snap.Changes {
					op := "upsert"
					if c.Kind == firestore.DocumentRemoved {
						op = "delete"
					}
					emit(model.Change{Kind: kind, ID: c.Doc.Ref.ID, Op: op})
				}
			}
		})
	}
	wg.Wait()
	if firstErr != nil && !errors.Is(firstErr, iterator.Done) {
		return mapErr("watch trip", firstErr)
	}
	return nil
}
