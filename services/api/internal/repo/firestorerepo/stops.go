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

// Stops implements repo.Stops.
type Stops struct {
	client *firestore.Client
}

var _ repo.Stops = (*Stops)(nil)

// NewStops returns the Firestore-backed stop store.
func NewStops(client *firestore.Client) *Stops { return &Stops{client: client} }

type placeDoc struct {
	Kind    string  `firestore:"kind"`
	PlaceID string  `firestore:"placeId,omitempty"`
	Name    string  `firestore:"name,omitempty"`
	Lat     float64 `firestore:"lat,omitempty"`
	Lng     float64 `firestore:"lng,omitempty"`
}

type stopDoc struct {
	Date        string     `firestore:"date"`
	Order       string     `firestore:"order"`
	Place       placeDoc   `firestore:"place"`
	StartLocal  string     `firestore:"startLocal,omitempty"`
	DurationMin int        `firestore:"durationMin,omitempty"`
	TimeZone    string     `firestore:"timeZone"`
	Note        string     `firestore:"note,omitempty"`
	DeletedAt   *time.Time `firestore:"deletedAt,omitempty"`
	CreatedAt   time.Time  `firestore:"createdAt,serverTimestamp"`
	UpdatedAt   time.Time  `firestore:"updatedAt,serverTimestamp"`
}

func (s *Stops) ref(tripID, stopID string) *firestore.DocumentRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(stopsCollection).Doc(stopID)
}

func stopToDoc(st model.Stop) stopDoc {
	p := placeDoc{Kind: "catalog", PlaceID: st.Place.PlaceID}
	if st.Place.Custom() {
		p = placeDoc{Kind: "custom", Name: st.Place.Name, Lat: st.Place.Lat, Lng: st.Place.Lng}
	}
	return stopDoc{Date: st.Date, Order: st.Order, Place: p, StartLocal: st.StartLocal, DurationMin: st.DurationMin,
		TimeZone: st.TimeZone, Note: st.Note}
}

func stopFromSnap(snap *firestore.DocumentSnapshot, tripID string) (model.Stop, error) {
	var d stopDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Stop{}, mapErr("decode stop", err)
	}
	return model.Stop{
		ID: snap.Ref.ID, TripID: tripID, Date: d.Date, Order: d.Order,
		Place:      model.PlaceRef{PlaceID: d.Place.PlaceID, Name: d.Place.Name, Lat: d.Place.Lat, Lng: d.Place.Lng},
		StartLocal: d.StartLocal, DurationMin: d.DurationMin, TimeZone: d.TimeZone, Note: d.Note,
		DeletedAt: d.DeletedAt, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}, nil
}

// Create stores a new stop.
func (s *Stops) Create(ctx context.Context, st model.Stop) (model.Stop, error) {
	if err := st.Validate(); err != nil {
		return model.Stop{}, err
	}
	if _, err := s.ref(st.TripID, st.ID).Create(ctx, stopToDoc(st)); err != nil {
		return model.Stop{}, mapErr("create stop", err)
	}
	return s.Get(ctx, st.TripID, st.ID)
}

// Get returns one stop, deleted or not.
func (s *Stops) Get(ctx context.Context, tripID, stopID string) (model.Stop, error) {
	if tripID == "" || stopID == "" {
		return model.Stop{}, repo.ErrNotFound
	}
	snap, err := s.ref(tripID, stopID).Get(ctx)
	if err != nil {
		return model.Stop{}, mapErr("get stop", err)
	}
	return stopFromSnap(snap, tripID)
}

// List returns the trip's live stops sorted by (date, order). It sorts here
// rather than in the query so no composite index is needed.
func (s *Stops) List(ctx context.Context, tripID string) ([]model.Stop, error) {
	iter := s.client.Collection(tripsCollection).Doc(tripID).Collection(stopsCollection).Documents(ctx)
	defer iter.Stop()
	var out []model.Stop
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list stops", err)
		}
		st, err := stopFromSnap(snap, tripID)
		if err != nil {
			return nil, err
		}
		if st.DeletedAt == nil {
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Update changes only the fields set in the patch.
func (s *Stops) Update(ctx context.Context, tripID, stopID string, p model.StopPatch) (model.Stop, error) {
	if err := p.Validate(); err != nil {
		return model.Stop{}, err
	}
	ups := []firestore.Update{{Path: "updatedAt", Value: firestore.ServerTimestamp}}
	add := func(path string, v any) { ups = append(ups, firestore.Update{Path: path, Value: v}) }
	if p.Date != nil {
		add("date", *p.Date)
	}
	if p.Order != nil {
		add("order", *p.Order)
	}
	if p.StartLocal != nil {
		add("startLocal", orDelete(*p.StartLocal, *p.StartLocal == ""))
	}
	if p.DurationMin != nil {
		add("durationMin", orDelete(*p.DurationMin, *p.DurationMin == 0))
	}
	if p.Note != nil {
		add("note", orDelete(*p.Note, *p.Note == ""))
	}
	if p.Deleted {
		add("deletedAt", firestore.ServerTimestamp)
	}
	if _, err := s.ref(tripID, stopID).Update(ctx, ups); err != nil {
		return model.Stop{}, mapErr("update stop", err)
	}
	return s.Get(ctx, tripID, stopID)
}

func orDelete(v any, empty bool) any {
	if empty {
		return firestore.Delete
	}
	return v
}
