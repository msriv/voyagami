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

const bookingsCollection = "bookings"

// Bookings implements repo.Bookings.
type Bookings struct {
	client *firestore.Client
}

var _ repo.Bookings = (*Bookings)(nil)

// NewBookings returns the Firestore-backed booking store.
func NewBookings(client *firestore.Client) *Bookings { return &Bookings{client: client} }

type bookingPlaceDoc struct {
	Code string `firestore:"code,omitempty"`
	Name string `firestore:"name"`
}

type bookingDoc struct {
	Kind          string           `firestore:"kind"`
	Title         string           `firestore:"title"`
	Provider      string           `firestore:"provider,omitempty"`
	StartsAt      *time.Time       `firestore:"startsAt,omitempty"`
	EndsAt        *time.Time       `firestore:"endsAt,omitempty"`
	StartTimeZone string           `firestore:"startTimeZone,omitempty"`
	EndTimeZone   string           `firestore:"endTimeZone,omitempty"`
	From          *bookingPlaceDoc `firestore:"from,omitempty"`
	To            *bookingPlaceDoc `firestore:"to,omitempty"`
	Price         *moneyDoc        `firestore:"price,omitempty"`
	Source        string           `firestore:"source"`
	CreatedBy     string           `firestore:"createdBy"`
	CreatedAt     time.Time        `firestore:"createdAt,serverTimestamp"`
}

func (s *Bookings) col(tripID string) *firestore.CollectionRef {
	return s.client.Collection(tripsCollection).Doc(tripID).Collection(bookingsCollection)
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func placeToDoc(p *model.BookingPlace) *bookingPlaceDoc {
	if p == nil {
		return nil
	}
	return &bookingPlaceDoc{Code: p.Code, Name: p.Name}
}

func placeFromDoc(p *bookingPlaceDoc) *model.BookingPlace {
	if p == nil {
		return nil
	}
	return &model.BookingPlace{Code: p.Code, Name: p.Name}
}

func bookingToDoc(b model.Booking) bookingDoc {
	d := bookingDoc{
		Kind: b.Kind, Title: b.Title, Provider: b.Provider, StartsAt: timePtr(b.StartsAt), EndsAt: timePtr(b.EndsAt),
		StartTimeZone: b.StartTimeZone, EndTimeZone: b.EndTimeZone, From: placeToDoc(b.From), To: placeToDoc(b.To),
		Source: "manual", CreatedBy: b.CreatedBy,
	}
	if b.Price != nil {
		d.Price = &moneyDoc{AmountMinor: b.Price.AmountMinor, Currency: b.Price.Currency}
	}
	return d
}

func bookingFromSnap(snap *firestore.DocumentSnapshot) (model.Booking, error) {
	var d bookingDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Booking{}, mapErr("decode booking", err)
	}
	b := model.Booking{
		ID: snap.Ref.ID, Kind: d.Kind, Title: d.Title, Provider: d.Provider,
		StartTimeZone: d.StartTimeZone, EndTimeZone: d.EndTimeZone,
		From: placeFromDoc(d.From), To: placeFromDoc(d.To), CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt,
	}
	if d.StartsAt != nil {
		b.StartsAt = *d.StartsAt
	}
	if d.EndsAt != nil {
		b.EndsAt = *d.EndsAt
	}
	if d.Price != nil {
		b.Price = &model.Money{AmountMinor: d.Price.AmountMinor, Currency: d.Price.Currency}
	}
	return b, nil
}

// Create stores a new booking.
func (s *Bookings) Create(ctx context.Context, tripID string, b model.Booking) (model.Booking, error) {
	if err := b.Validate(); err != nil {
		return model.Booking{}, err
	}
	if _, err := s.col(tripID).Doc(b.ID).Create(ctx, bookingToDoc(b)); err != nil {
		return model.Booking{}, mapErr("create booking", err)
	}
	return s.Get(ctx, tripID, b.ID)
}

// Get returns one booking.
func (s *Bookings) Get(ctx context.Context, tripID, id string) (model.Booking, error) {
	snap, err := s.col(tripID).Doc(id).Get(ctx)
	if err != nil {
		return model.Booking{}, mapErr("get booking", err)
	}
	return bookingFromSnap(snap)
}

// List returns bookings soonest first, undated last.
func (s *Bookings) List(ctx context.Context, tripID string) ([]model.Booking, error) {
	iter := s.col(tripID).Documents(ctx)
	defer iter.Stop()
	var out []model.Booking
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list bookings", err)
		}
		b, err := bookingFromSnap(snap)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.StartsAt.IsZero() != b.StartsAt.IsZero() {
			return !a.StartsAt.IsZero()
		}
		if !a.StartsAt.Equal(b.StartsAt) {
			return a.StartsAt.Before(b.StartsAt)
		}
		return a.ID < b.ID
	})
	return out, nil
}

// Replace overwrites the content fields and leaves who made it and when alone.
func (s *Bookings) Replace(ctx context.Context, tripID string, b model.Booking) (model.Booking, error) {
	if err := b.Validate(); err != nil {
		return model.Booking{}, err
	}
	d := bookingToDoc(b)
	field := func(path string, v any, empty bool) firestore.Update {
		if empty {
			return firestore.Update{Path: path, Value: firestore.Delete}
		}
		return firestore.Update{Path: path, Value: v}
	}
	ups := []firestore.Update{
		{Path: "kind", Value: d.Kind},
		{Path: "title", Value: d.Title},
		field("provider", d.Provider, d.Provider == ""),
		field("startsAt", d.StartsAt, d.StartsAt == nil),
		field("endsAt", d.EndsAt, d.EndsAt == nil),
		field("startTimeZone", d.StartTimeZone, d.StartTimeZone == ""),
		field("endTimeZone", d.EndTimeZone, d.EndTimeZone == ""),
		field("from", d.From, d.From == nil),
		field("to", d.To, d.To == nil),
		field("price", d.Price, d.Price == nil),
	}
	if _, err := s.col(tripID).Doc(b.ID).Update(ctx, ups); err != nil {
		return model.Booking{}, mapErr("replace booking", err)
	}
	return s.Get(ctx, tripID, b.ID)
}

// Delete removes a booking.
func (s *Bookings) Delete(ctx context.Context, tripID, id string) error {
	_, err := s.col(tripID).Doc(id).Delete(ctx)
	return mapErr("delete booking", err)
}
