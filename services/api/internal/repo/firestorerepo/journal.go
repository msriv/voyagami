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

const journalCollection = "journal"

// Journal implements repo.Journal.
type Journal struct {
	client *firestore.Client
}

var _ repo.Journal = (*Journal)(nil)

// NewJournal returns the Firestore-backed journal store.
func NewJournal(client *firestore.Client) *Journal { return &Journal{client: client} }

type journalDoc struct {
	Date       string    `firestore:"date"`
	AuthorUID  string    `firestore:"authorUid"`
	Visibility string    `firestore:"visibility"`
	Text       string    `firestore:"text"`
	Mood       string    `firestore:"mood,omitempty"`
	StopIDs    []string  `firestore:"stopIds,omitempty"`
	CreatedAt  time.Time `firestore:"createdAt"`
	UpdatedAt  time.Time `firestore:"updatedAt"`
}

func (s *Journal) ref(tripID, id string) *firestore.DocumentRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(journalCollection).Doc(id)
}

func journalFromDoc(d journalDoc) model.JournalEntry {
	return model.JournalEntry{
		Date: d.Date, AuthorUID: d.AuthorUID, Visibility: d.Visibility, Text: d.Text, Mood: d.Mood,
		StopIDs: d.StopIDs, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

// Put writes the entry in one transaction so the creation time survives edits.
func (s *Journal) Put(ctx context.Context, tripID string, e model.JournalEntry) (model.JournalEntry, error) {
	if err := e.Validate(); err != nil {
		return model.JournalEntry{}, err
	}
	ref := s.ref(tripID, e.ID())
	now := time.Now().UTC().Truncate(time.Microsecond) // Firestore keeps microseconds
	d := journalDoc{
		Date: e.Date, AuthorUID: e.AuthorUID, Visibility: e.Visibility, Text: e.Text, Mood: e.Mood,
		StopIDs: e.StopIDs, CreatedAt: now, UpdatedAt: now,
	}
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err == nil {
			var old journalDoc
			if err := snap.DataTo(&old); err != nil {
				return err
			}
			d.CreatedAt = old.CreatedAt
		} else if !isNotFound(err) {
			return err
		}
		return tx.Set(ref, d)
	})
	if err != nil {
		return model.JournalEntry{}, mapErr("put journal entry", err)
	}
	return journalFromDoc(d), nil
}

// List returns every entry, newest day first, then by author.
func (s *Journal) List(ctx context.Context, tripID string) ([]model.JournalEntry, error) {
	iter := s.client.Collection(tripsCollection).Doc(tripID).Collection(journalCollection).Documents(ctx)
	defer iter.Stop()
	var out []model.JournalEntry
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list journal", err)
		}
		var d journalDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode journal entry", err)
		}
		out = append(out, journalFromDoc(d))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date > out[j].Date
		}
		return out[i].AuthorUID < out[j].AuthorUID
	})
	return out, nil
}

// Delete removes one entry.
func (s *Journal) Delete(ctx context.Context, tripID, date, authorUID string) error {
	_, err := s.ref(tripID, date+"_"+authorUID).Delete(ctx)
	return mapErr("delete journal entry", err)
}
