package firestorerepo

import (
	"context"
	"errors"
	"sort"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Trips implements repo.Trips.
type Trips struct {
	client *firestore.Client
}

var _ repo.Trips = (*Trips)(nil)

// NewTrips returns the Firestore-backed trip store.
func NewTrips(client *firestore.Client) *Trips { return &Trips{client: client} }

func (s *Trips) tripRef(id string) *firestore.DocumentRef {
	return s.client.Collection(tripsCollection).Doc(id)
}

func (s *Trips) memberRef(tripID, uid string) *firestore.DocumentRef {
	return s.tripRef(tripID).Collection(membersCollection).Doc(uid)
}

// Create stores the trip and its owner in one transaction.
func (s *Trips) Create(ctx context.Context, t model.Trip, owner model.Member) (model.Trip, error) {
	owner.Role = model.RoleOwner
	owner.Status = model.MemberActive
	t.OwnerUID = owner.UID
	t.MemberUIDs = []string{owner.UID}
	if err := t.Validate(); err != nil {
		return model.Trip{}, err
	}
	if err := owner.Validate(); err != nil {
		return model.Trip{}, err
	}
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		if err := tx.Create(s.tripRef(t.ID), tripToDoc(t)); err != nil {
			return err
		}
		return tx.Create(s.memberRef(t.ID, owner.UID), memberToDoc(owner))
	})
	if err != nil {
		return model.Trip{}, mapErr("create trip", err)
	}
	return s.Get(ctx, t.ID)
}

// Get returns one trip.
func (s *Trips) Get(ctx context.Context, id string) (model.Trip, error) {
	if id == "" {
		return model.Trip{}, repo.ErrNotFound
	}
	snap, err := s.tripRef(id).Get(ctx)
	if err != nil {
		return model.Trip{}, mapErr("get trip", err)
	}
	return decodeTrip(snap)
}

func decodeTrip(snap *firestore.DocumentSnapshot) (model.Trip, error) {
	var d tripDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Trip{}, mapErr("decode trip", err)
	}
	return tripFromDoc(snap.Ref.ID, d), nil
}

// ListForMember returns the trips an active member is on, soonest first.
func (s *Trips) ListForMember(ctx context.Context, uid string) ([]model.Trip, error) {
	// No server-side ordering: trips with flexible dates have no dates.start
	// and an ordered query would silently skip them.
	iter := s.client.Collection(tripsCollection).Where("memberUids", "array-contains", uid).Documents(ctx)
	defer iter.Stop()
	var trips []model.Trip
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list trips", err)
		}
		t, err := decodeTrip(snap)
		if err != nil {
			return nil, err
		}
		trips = append(trips, t)
	}
	sort.SliceStable(trips, func(i, j int) bool {
		a, b := trips[i], trips[j]
		if (a.Dates.Start == "") != (b.Dates.Start == "") {
			return a.Dates.Start != "" // dated trips first
		}
		if a.Dates.Start != b.Dates.Start {
			return a.Dates.Start < b.Dates.Start
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.ID < b.ID
	})
	return trips, nil
}

// Update changes only the fields set in the patch.
func (s *Trips) Update(ctx context.Context, id string, p model.TripPatch) (model.Trip, error) {
	if err := p.Validate(); err != nil {
		return model.Trip{}, err
	}
	ups := []firestore.Update{{Path: "updatedAt", Value: firestore.ServerTimestamp}}
	add := func(path string, v any) { ups = append(ups, firestore.Update{Path: path, Value: v}) }
	if p.Title != nil {
		add("title", *p.Title)
	}
	if p.Destinations != nil {
		add("destinations", destinationsToDoc(*p.Destinations))
		add("destinationCountries", countriesOf(*p.Destinations))
	}
	if p.Dates != nil {
		add("dates", datesToDoc(*p.Dates))
	}
	if p.TimeZone != nil {
		add("timeZone", *p.TimeZone)
	}
	if p.Budget != nil {
		add("budget", budgetToDoc(*p.Budget))
	}
	if p.Status != nil {
		add("status", string(*p.Status))
	}
	if _, err := s.tripRef(id).Update(ctx, ups); err != nil {
		return model.Trip{}, mapErr("update trip", err)
	}
	return s.Get(ctx, id)
}

// Member returns one member, including people who left.
func (s *Trips) Member(ctx context.Context, tripID, uid string) (model.Member, error) {
	if tripID == "" || uid == "" {
		return model.Member{}, repo.ErrNotFound
	}
	snap, err := s.memberRef(tripID, uid).Get(ctx)
	if err != nil {
		return model.Member{}, mapErr("get member", err)
	}
	var d memberDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Member{}, mapErr("decode member", err)
	}
	return memberFromDoc(d), nil
}

// Members returns everyone with a member document, oldest first.
func (s *Trips) Members(ctx context.Context, tripID string) ([]model.Member, error) {
	iter := s.tripRef(tripID).Collection(membersCollection).Documents(ctx)
	defer iter.Stop()
	var out []model.Member
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list members", err)
		}
		var d memberDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode member", err)
		}
		out = append(out, memberFromDoc(d))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].JoinedAt.Equal(out[j].JoinedAt) {
			return out[i].JoinedAt.Before(out[j].JoinedAt)
		}
		return out[i].UID < out[j].UID
	})
	return out, nil
}

// AddMember adds or reactivates an editor or viewer.
func (s *Trips) AddMember(ctx context.Context, tripID string, m model.Member) error {
	m.Status = model.MemberActive
	if err := m.Validate(); err != nil {
		return err
	}
	if m.Role == model.RoleOwner {
		return repo.ErrForbidden
	}
	tripRef, memberRef := s.tripRef(tripID), s.memberRef(tripID, m.UID)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(tripRef); err != nil {
			return err
		}
		snap, err := tx.Get(memberRef)
		if err == nil {
			var d memberDoc
			if err := snap.DataTo(&d); err != nil {
				return err
			}
			if d.Status == string(model.MemberActive) {
				return repo.ErrAlreadyExists
			}
		} else if !isNotFound(err) {
			return err
		}
		if err := tx.Set(memberRef, memberToDoc(m)); err != nil {
			return err
		}
		return tx.Update(tripRef, []firestore.Update{
			{Path: "memberUids", Value: firestore.ArrayUnion(m.UID)},
			{Path: "updatedAt", Value: firestore.ServerTimestamp},
		})
	})
	return mapErr("add member", err)
}

// SetMemberRole changes an active editor or viewer between editor and viewer.
func (s *Trips) SetMemberRole(ctx context.Context, tripID, uid string, role model.Role) error {
	if role != model.RoleEditor && role != model.RoleViewer {
		return repo.ErrForbidden
	}
	ref := s.memberRef(tripID, uid)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var d memberDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		if d.Status != string(model.MemberActive) {
			return repo.ErrNotFound
		}
		if d.Role == string(model.RoleOwner) {
			return repo.ErrForbidden
		}
		return tx.Update(ref, []firestore.Update{{Path: "role", Value: string(role)}})
	})
	return mapErr("set member role", err)
}

// EndMembership marks a member left or removed. The owner cannot be removed.
func (s *Trips) EndMembership(ctx context.Context, tripID, uid string, st model.MemberStatus) error {
	if st != model.MemberLeft && st != model.MemberRemoved {
		return &model.ValidationError{Problems: []string{"status must be left or removed"}}
	}
	tripRef, ref := s.tripRef(tripID), s.memberRef(tripID, uid)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var d memberDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		if d.Role == string(model.RoleOwner) {
			return repo.ErrForbidden
		}
		if err := tx.Update(ref, []firestore.Update{{Path: "status", Value: string(st)}}); err != nil {
			return err
		}
		return tx.Update(tripRef, []firestore.Update{
			{Path: "memberUids", Value: firestore.ArrayRemove(uid)},
			{Path: "updatedAt", Value: firestore.ServerTimestamp},
		})
	})
	return mapErr("end membership", err)
}
