package firestorerepo

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const (
	sponsoredCollection      = "sponsoredCards"
	sponsoredStatsCollection = "sponsoredStats"
	hiddenCardsCollection    = "hiddenCards"
	seenCardsCollection      = "sponsoredSeen"
)

// Sponsored implements repo.Sponsored. The inventory is the sponsoredCards
// collection, filled by hand in the console; no ad network is involved.
type Sponsored struct {
	client *firestore.Client
}

var _ repo.Sponsored = (*Sponsored)(nil)

// NewSponsored returns the Firestore-backed sponsored card store.
func NewSponsored(client *firestore.Client) *Sponsored { return &Sponsored{client: client} }

type sponsoredDoc struct {
	Active    bool      `firestore:"active"`
	Sponsor   string    `firestore:"sponsor"`
	Title     string    `firestore:"title"`
	Body      string    `firestore:"body"`
	ImageURL  string    `firestore:"imageUrl"`
	URL       string    `firestore:"url"`
	Countries []string  `firestore:"countries"`
	Groups    []string  `firestore:"groups"`
	Priority  int       `firestore:"priority"`
	StartsAt  time.Time `firestore:"startsAt"`
	EndsAt    time.Time `firestore:"endsAt"`
}

func (d sponsoredDoc) card(id string) model.SponsoredCard {
	c := model.SponsoredCard{ID: id, Sponsor: d.Sponsor, Title: d.Title, Body: d.Body, ImageURL: d.ImageURL, URL: d.URL,
		Countries: d.Countries, Priority: d.Priority, StartsAt: d.StartsAt, EndsAt: d.EndsAt}
	for _, g := range d.Groups {
		c.Groups = append(c.Groups, model.PlaceGroup(g))
	}
	return c
}

// Live returns the active cards that run at now.
//
// shortcut: reads every active card and filters in memory; fine for a hand-kept
// table of a few dozen deals, move to a targeted query when it grows.
func (s *Sponsored) Live(ctx context.Context, now time.Time) ([]model.SponsoredCard, error) {
	iter := s.client.Collection(sponsoredCollection).Where("active", "==", true).Documents(ctx)
	defer iter.Stop()
	var out []model.SponsoredCard
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, mapErr("list sponsored cards", err)
		}
		var d sponsoredDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode sponsored card", err)
		}
		if c := d.card(snap.Ref.ID); c.LiveAt(now) {
			out = append(out, c)
		}
	}
}

// Get returns one card.
func (s *Sponsored) Get(ctx context.Context, id string) (model.SponsoredCard, error) {
	snap, err := s.client.Collection(sponsoredCollection).Doc(id).Get(ctx)
	if err != nil {
		return model.SponsoredCard{}, mapErr("get sponsored card", err)
	}
	var d sponsoredDoc
	if err := snap.DataTo(&d); err != nil {
		return model.SponsoredCard{}, mapErr("decode sponsored card", err)
	}
	return d.card(id), nil
}

func (s *Sponsored) hiddenColl(uid string) *firestore.CollectionRef {
	return s.client.Collection(usersCollection).Doc(uid).Collection(hiddenCardsCollection)
}

// Hidden returns the cards still hidden at now.
func (s *Sponsored) Hidden(ctx context.Context, uid string, now time.Time) (map[string]bool, error) {
	docs, err := s.hiddenColl(uid).Where("until", ">", now).Documents(ctx).GetAll()
	if err != nil {
		return nil, mapErr("list hidden cards", err)
	}
	out := make(map[string]bool, len(docs))
	for _, d := range docs {
		out[d.Ref.ID] = true
	}
	return out, nil
}

// Hide hides a card until the given time.
func (s *Sponsored) Hide(ctx context.Context, uid, cardID string, until time.Time) error {
	_, err := s.hiddenColl(uid).Doc(cardID).Set(ctx, map[string]any{"until": until.UTC()})
	return mapErr("hide sponsored card", err)
}

// Record counts one impression or click per person, card, kind and day. The
// marker lives under the person, so deleting the account removes it.
func (s *Sponsored) Record(ctx context.Context, uid, cardID string, kind model.SponsoredEvent, now time.Time) error {
	day := now.UTC().Format("2006-01-02")
	marker := s.client.Collection(usersCollection).Doc(uid).Collection(seenCardsCollection).Doc(day + "_" + cardID + "_" + string(kind))
	stats := s.client.Collection(sponsoredStatsCollection).Doc(cardID + "_" + day)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(marker); err == nil {
			return nil // already counted today
		} else if !isNotFound(err) {
			return err
		}
		if err := tx.Create(marker, map[string]any{"at": now.UTC()}); err != nil {
			return err
		}
		return tx.Set(stats, map[string]any{"cardId": cardID, "day": day, string(kind) + "s": firestore.Increment(1)}, firestore.MergeAll)
	})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	return mapErr("record sponsored event", err)
}
