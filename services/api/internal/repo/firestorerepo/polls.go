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

const (
	pollsCollection = "polls"
	votesCollection = "votes"
)

// Polls implements repo.Polls.
type Polls struct {
	client *firestore.Client
}

var _ repo.Polls = (*Polls)(nil)

// NewPolls returns the Firestore-backed poll store.
func NewPolls(client *firestore.Client) *Polls { return &Polls{client: client} }

type pollOptionDoc struct {
	ID    string    `firestore:"id"`
	Label string    `firestore:"label"`
	Place *placeDoc `firestore:"place,omitempty"`
}

type pollDoc struct {
	Question       string          `firestore:"question"`
	Options        []pollOptionDoc `firestore:"options"`
	Multi          bool            `firestore:"multi"`
	Status         string          `firestore:"status"`
	ClosesAt       time.Time       `firestore:"closesAt,omitempty"`
	ClosedAt       time.Time       `firestore:"closedAt,omitempty"`
	ResultOptionID string          `firestore:"resultOptionId,omitempty"`
	ResultCounts   map[string]int  `firestore:"resultCounts,omitempty"`
	DecidedBy      string          `firestore:"decidedBy,omitempty"`
	CreatedBy      string          `firestore:"createdBy"`
	CreatedAt      time.Time       `firestore:"createdAt"`
}

type voteDoc struct {
	OptionIDs []string  `firestore:"optionIds"`
	VotedAt   time.Time `firestore:"votedAt"`
}

func (s *Polls) ref(tripID, id string) *firestore.DocumentRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(pollsCollection).Doc(id)
}

func pollToDoc(p model.Poll) pollDoc {
	d := pollDoc{
		Question: p.Question, Multi: p.Multi, Status: p.Status, ClosesAt: micro(p.ClosesAt), ClosedAt: micro(p.ClosedAt),
		CreatedBy: p.CreatedBy, CreatedAt: micro(p.CreatedAt),
	}
	for _, o := range p.Options {
		od := pollOptionDoc{ID: o.ID, Label: o.Label}
		if o.Place != nil {
			pd := placeDoc{Kind: "catalog", PlaceID: o.Place.PlaceID}
			if o.Place.Custom() {
				pd = placeDoc{Kind: "custom", Name: o.Place.Name, Lat: o.Place.Lat, Lng: o.Place.Lng, Address: o.Place.Address, Category: o.Place.Category}
			}
			od.Place = &pd
		}
		d.Options = append(d.Options, od)
	}
	if r := p.Result; r != nil {
		d.ResultOptionID, d.ResultCounts, d.DecidedBy = r.OptionID, r.Counts, r.DecidedBy
	}
	return d
}

func micro(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC().Truncate(time.Microsecond)
}

func pollFromDoc(id string, d pollDoc) model.Poll {
	p := model.Poll{
		ID: id, Question: d.Question, Multi: d.Multi, Status: d.Status, ClosesAt: d.ClosesAt, ClosedAt: d.ClosedAt,
		CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt,
	}
	for _, od := range d.Options {
		o := model.PollOption{ID: od.ID, Label: od.Label}
		if pd := od.Place; pd != nil {
			o.Place = &model.PlaceRef{PlaceID: pd.PlaceID, Name: pd.Name, Lat: pd.Lat, Lng: pd.Lng, Address: pd.Address, Category: pd.Category}
		}
		p.Options = append(p.Options, o)
	}
	if d.Status == model.PollClosed {
		p.Result = &model.PollResult{OptionID: d.ResultOptionID, Counts: d.ResultCounts, DecidedBy: d.DecidedBy}
	}
	return p
}

// Create stores the poll as open.
func (s *Polls) Create(ctx context.Context, tripID string, p model.Poll) (model.Poll, error) {
	p.Status, p.Result, p.ClosedAt = model.PollOpen, nil, time.Time{}
	p.CreatedAt = micro(time.Now())
	if err := p.Validate(); err != nil {
		return model.Poll{}, err
	}
	if _, err := s.ref(tripID, p.ID).Create(ctx, pollToDoc(p)); err != nil {
		return model.Poll{}, mapErr("create poll", err)
	}
	return s.Get(ctx, tripID, p.ID)
}

// Get returns one poll.
func (s *Polls) Get(ctx context.Context, tripID, id string) (model.Poll, error) {
	if tripID == "" || id == "" {
		return model.Poll{}, repo.ErrNotFound
	}
	snap, err := s.ref(tripID, id).Get(ctx)
	if err != nil {
		return model.Poll{}, mapErr("get poll", err)
	}
	var d pollDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Poll{}, mapErr("decode poll", err)
	}
	return pollFromDoc(id, d), nil
}

// List returns the trip's polls, newest first.
func (s *Polls) List(ctx context.Context, tripID string) ([]model.Poll, error) {
	iter := s.client.Collection(tripsCollection).Doc(tripID).Collection(pollsCollection).Documents(ctx)
	defer iter.Stop()
	var out []model.Poll
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list polls", err)
		}
		var d pollDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode poll", err)
		}
		out = append(out, pollFromDoc(snap.Ref.ID, d))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Vote stores or replaces a vote while the poll is open.
func (s *Polls) Vote(ctx context.Context, tripID, pollID string, v model.Vote, now time.Time) error {
	ref := s.ref(tripID, pollID)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var d pollDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		p := pollFromDoc(pollID, d)
		if !p.Open(now) {
			return repo.ErrConflict
		}
		if err := v.Validate(p); err != nil {
			return err
		}
		return tx.Set(ref.Collection(votesCollection).Doc(v.UID), voteDoc{OptionIDs: v.OptionIDs, VotedAt: micro(now)})
	})
	return mapErr("vote", err)
}

// Votes returns every vote on the poll, ordered by voter.
func (s *Polls) Votes(ctx context.Context, tripID, pollID string) ([]model.Vote, error) {
	iter := s.ref(tripID, pollID).Collection(votesCollection).Documents(ctx)
	defer iter.Stop()
	var out []model.Vote
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list votes", err)
		}
		var d voteDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode vote", err)
		}
		out = append(out, model.Vote{UID: snap.Ref.ID, OptionIDs: d.OptionIDs, VotedAt: d.VotedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UID < out[j].UID })
	return out, nil
}

// Close counts the votes and ends the poll.
func (s *Polls) Close(ctx context.Context, tripID, pollID, chosen string, now time.Time) (model.Poll, error) {
	ref := s.ref(tripID, pollID)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var d pollDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		if d.Status == model.PollClosed {
			return nil
		}
		p := pollFromDoc(pollID, d)
		docs, err := tx.Documents(ref.Collection(votesCollection)).GetAll()
		if err != nil {
			return err
		}
		votes := make([]model.Vote, 0, len(docs))
		for _, vs := range docs {
			var vd voteDoc
			if err := vs.DataTo(&vd); err != nil {
				return err
			}
			votes = append(votes, model.Vote{UID: vs.Ref.ID, OptionIDs: vd.OptionIDs})
		}
		res, err := model.Tally(p, votes, chosen)
		if err != nil {
			return err
		}
		if res.OptionID == "" {
			return model.ErrPollNoWinner
		}
		return tx.Update(ref, []firestore.Update{
			{Path: "status", Value: model.PollClosed},
			{Path: "closedAt", Value: micro(now)},
			{Path: "resultOptionId", Value: res.OptionID},
			{Path: "resultCounts", Value: res.Counts},
			{Path: "decidedBy", Value: res.DecidedBy},
		})
	})
	if err != nil {
		return model.Poll{}, mapErr("close poll", err)
	}
	return s.Get(ctx, tripID, pollID)
}
