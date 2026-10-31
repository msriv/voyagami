package firestorerepo

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

const stayClicksCollection = "stayClicks"

// StayClicks keeps a count of taps per partner and day in stayClicks/{partner}_{day}.
// No person, trip or place is stored.
type StayClicks struct {
	client *firestore.Client
}

// NewStayClicks returns the Firestore-backed click counter.
func NewStayClicks(client *firestore.Client) *StayClicks { return &StayClicks{client: client} }

// Add counts one tap for the partner on the UTC day of the given time.
func (s *StayClicks) Add(ctx context.Context, partnerID string, at time.Time) error {
	day := at.UTC().Format("2006-01-02")
	_, err := s.client.Collection(stayClicksCollection).Doc(partnerID+"_"+day).Set(ctx, map[string]any{
		"partnerId": partnerID, "day": day, "count": firestore.Increment(1),
	}, firestore.MergeAll)
	return mapErr("count stay click", err)
}

// Count returns the taps recorded for the partner on a day. Used by tests and reports.
func (s *StayClicks) Count(ctx context.Context, partnerID string, at time.Time) (int, error) {
	snap, err := s.client.Collection(stayClicksCollection).Doc(partnerID + "_" + at.UTC().Format("2006-01-02")).Get(ctx)
	if isNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, mapErr("get stay clicks", err)
	}
	var d struct {
		Count int `firestore:"count"`
	}
	if err := snap.DataTo(&d); err != nil {
		return 0, mapErr("decode stay clicks", err)
	}
	return d.Count, nil
}
