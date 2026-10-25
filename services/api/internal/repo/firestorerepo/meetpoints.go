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
	meetPointsCollection = "meetPoints"
	presenceCollection   = "presence"
)

// MeetPoints implements repo.MeetPoints.
type MeetPoints struct {
	client *firestore.Client
}

var _ repo.MeetPoints = (*MeetPoints)(nil)

// NewMeetPoints returns the Firestore-backed meet point store.
func NewMeetPoints(client *firestore.Client) *MeetPoints { return &MeetPoints{client: client} }

type meetDoc struct {
	Name        string    `firestore:"name"`
	Place       placeDoc  `firestore:"place"`
	At          time.Time `firestore:"at,omitempty"`
	CreatedBy   string    `firestore:"createdBy"`
	CreatedAt   time.Time `firestore:"createdAt"`
	ConfirmedBy []string  `firestore:"confirmedBy"`
}

func (s *MeetPoints) ref(tripID, id string) *firestore.DocumentRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(meetPointsCollection).Doc(id)
}

func placeDocOf(p model.PlaceRef) placeDoc {
	if p.Custom() {
		return placeDoc{Kind: "custom", Name: p.Name, Lat: p.Lat, Lng: p.Lng, Address: p.Address, Category: p.Category}
	}
	return placeDoc{Kind: "catalog", PlaceID: p.PlaceID}
}

func placeOfDoc(d placeDoc) model.PlaceRef {
	return model.PlaceRef{PlaceID: d.PlaceID, Name: d.Name, Lat: d.Lat, Lng: d.Lng, Address: d.Address, Category: d.Category}
}

func meetFromDoc(id string, d meetDoc) model.MeetPoint {
	return model.MeetPoint{ID: id, Name: d.Name, Place: placeOfDoc(d.Place), At: d.At, CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt, ConfirmedBy: d.ConfirmedBy}
}

// Create stores a meet point.
func (s *MeetPoints) Create(ctx context.Context, tripID string, m model.MeetPoint) (model.MeetPoint, error) {
	if err := m.Validate(); err != nil {
		return model.MeetPoint{}, err
	}
	if _, err := s.ref(tripID, m.ID).Create(ctx, meetDoc{
		Name: m.Name, Place: placeDocOf(m.Place), At: micro(m.At), CreatedBy: m.CreatedBy,
		CreatedAt: micro(time.Now()), ConfirmedBy: []string{},
	}); err != nil {
		return model.MeetPoint{}, mapErr("create meet point", err)
	}
	return s.Get(ctx, tripID, m.ID)
}

// Get returns one meet point.
func (s *MeetPoints) Get(ctx context.Context, tripID, id string) (model.MeetPoint, error) {
	if tripID == "" || id == "" {
		return model.MeetPoint{}, repo.ErrNotFound
	}
	snap, err := s.ref(tripID, id).Get(ctx)
	if err != nil {
		return model.MeetPoint{}, mapErr("get meet point", err)
	}
	var d meetDoc
	if err := snap.DataTo(&d); err != nil {
		return model.MeetPoint{}, mapErr("decode meet point", err)
	}
	return meetFromDoc(id, d), nil
}

// List returns the trip's meet points, soonest first, undated last.
func (s *MeetPoints) List(ctx context.Context, tripID string) ([]model.MeetPoint, error) {
	iter := s.client.Collection(tripsCollection).Doc(tripID).Collection(meetPointsCollection).Documents(ctx)
	defer iter.Stop()
	var out []model.MeetPoint
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list meet points", err)
		}
		var d meetDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode meet point", err)
		}
		out = append(out, meetFromDoc(snap.Ref.ID, d))
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.At.IsZero() != b.At.IsZero() {
			return !a.At.IsZero()
		}
		if !a.At.Equal(b.At) {
			return a.At.Before(b.At)
		}
		return a.ID < b.ID
	})
	return out, nil
}

// Update changes only the fields set in the patch.
func (s *MeetPoints) Update(ctx context.Context, tripID, id string, p model.MeetPointPatch) (model.MeetPoint, error) {
	if err := p.Validate(); err != nil {
		return model.MeetPoint{}, err
	}
	var ups []firestore.Update
	if p.Name != nil {
		ups = append(ups, firestore.Update{Path: "name", Value: *p.Name})
	}
	if p.Place != nil {
		ups = append(ups, firestore.Update{Path: "place", Value: placeDocOf(*p.Place)})
	}
	if p.At != nil {
		ups = append(ups, firestore.Update{Path: "at", Value: micro(*p.At)})
	}
	if len(ups) > 0 {
		if _, err := s.ref(tripID, id).Update(ctx, ups); err != nil {
			return model.MeetPoint{}, mapErr("update meet point", err)
		}
	}
	return s.Get(ctx, tripID, id)
}

// SetConfirmed adds or removes the person from those coming.
func (s *MeetPoints) SetConfirmed(ctx context.Context, tripID, id, uid string, coming bool) (model.MeetPoint, error) {
	var op any = firestore.ArrayRemove(uid)
	if coming {
		op = firestore.ArrayUnion(uid)
	}
	if _, err := s.ref(tripID, id).Update(ctx, []firestore.Update{{Path: "confirmedBy", Value: op}}); err != nil {
		return model.MeetPoint{}, mapErr("confirm meet point", err)
	}
	return s.Get(ctx, tripID, id)
}

// Delete removes a meet point.
func (s *MeetPoints) Delete(ctx context.Context, tripID, id string) error {
	_, err := s.ref(tripID, id).Delete(ctx)
	return mapErr("delete meet point", err)
}

// Presence implements repo.Presence.
type Presence struct {
	client *firestore.Client
}

var _ repo.Presence = (*Presence)(nil)

// NewPresence returns the Firestore-backed store for shared locations.
func NewPresence(client *firestore.Client) *Presence { return &Presence{client: client} }

type presenceDoc struct {
	Lat         float64   `firestore:"lat"`
	Lng         float64   `firestore:"lng"`
	AccuracyM   int       `firestore:"accuracyM"`
	MeetPointID string    `firestore:"meetPointId,omitempty"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
	// EndsAt is the field the Firestore TTL policy on presence watches.
	EndsAt time.Time `firestore:"endsAt"`
}

func (s *Presence) col(tripID string) *firestore.CollectionRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(presenceCollection)
}

// Put stores or replaces the person's shared location.
func (s *Presence) Put(ctx context.Context, tripID string, p model.Presence) error {
	if err := p.Validate(); err != nil {
		return err
	}
	_, err := s.col(tripID).Doc(p.UID).Set(ctx, presenceDoc{
		Lat: p.Lat, Lng: p.Lng, AccuracyM: p.AccuracyM, MeetPointID: p.MeetPointID,
		UpdatedAt: micro(p.UpdatedAt), EndsAt: micro(p.EndsAt),
	})
	return mapErr("share location", err)
}

// Get returns the person's entry, ended or not.
func (s *Presence) Get(ctx context.Context, tripID, uid string) (model.Presence, error) {
	if tripID == "" || uid == "" {
		return model.Presence{}, repo.ErrNotFound
	}
	snap, err := s.col(tripID).Doc(uid).Get(ctx)
	if err != nil {
		return model.Presence{}, mapErr("get shared location", err)
	}
	var d presenceDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Presence{}, mapErr("decode shared location", err)
	}
	return model.Presence{UID: uid, Lat: d.Lat, Lng: d.Lng, AccuracyM: d.AccuracyM, MeetPointID: d.MeetPointID, UpdatedAt: d.UpdatedAt, EndsAt: d.EndsAt}, nil
}

// Delete stops the person's sharing.
func (s *Presence) Delete(ctx context.Context, tripID, uid string) error {
	_, err := s.col(tripID).Doc(uid).Delete(ctx)
	return mapErr("stop sharing", err)
}

// List returns locations still shared at now and removes the ended ones.
func (s *Presence) List(ctx context.Context, tripID string, now time.Time) ([]model.Presence, error) {
	iter := s.col(tripID).Documents(ctx)
	defer iter.Stop()
	var out []model.Presence
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list shared locations", err)
		}
		var d presenceDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode shared location", err)
		}
		if !now.Before(d.EndsAt) {
			// Sharing has ended. The TTL policy would catch this later; do it now so it never lingers.
			_, _ = snap.Ref.Delete(ctx)
			continue
		}
		out = append(out, model.Presence{UID: snap.Ref.ID, Lat: d.Lat, Lng: d.Lng, AccuracyM: d.AccuracyM, MeetPointID: d.MeetPointID, UpdatedAt: d.UpdatedAt, EndsAt: d.EndsAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UID < out[j].UID })
	return out, nil
}
