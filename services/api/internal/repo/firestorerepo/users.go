package firestorerepo

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Users implements repo.Users.
type Users struct {
	client *firestore.Client
}

var _ repo.Users = (*Users)(nil)

// NewUsers returns the Firestore-backed user store.
func NewUsers(client *firestore.Client) *Users { return &Users{client: client} }

func (s *Users) doc(id string) *firestore.DocumentRef {
	return s.client.Collection(usersCollection).Doc(id)
}

// Create stores a new user.
func (s *Users) Create(ctx context.Context, u model.User) (model.User, error) {
	if err := u.Validate(); err != nil {
		return model.User{}, err
	}
	if _, err := s.doc(u.ID).Create(ctx, userToDoc(u)); err != nil {
		return model.User{}, mapErr("create user", err)
	}
	return s.Get(ctx, u.ID)
}

// Get returns one user.
func (s *Users) Get(ctx context.Context, id string) (model.User, error) {
	if id == "" {
		return model.User{}, repo.ErrNotFound
	}
	snap, err := s.doc(id).Get(ctx)
	if err != nil {
		return model.User{}, mapErr("get user", err)
	}
	var d userDoc
	if err := snap.DataTo(&d); err != nil {
		return model.User{}, mapErr("decode user", err)
	}
	return userFromDoc(snap.Ref.ID, d), nil
}

// Update changes only the fields set in the patch.
func (s *Users) Update(ctx context.Context, id string, p model.UserPatch) (model.User, error) {
	if err := p.Validate(); err != nil {
		return model.User{}, err
	}
	ups := []firestore.Update{{Path: "updatedAt", Value: firestore.ServerTimestamp}}
	add := func(path string, v any) { ups = append(ups, firestore.Update{Path: path, Value: v}) }
	if p.DisplayName != nil {
		add("displayName", *p.DisplayName)
	}
	if p.PhotoURL != nil {
		add("photoUrl", *p.PhotoURL)
	}
	if p.HomeRegion != nil {
		add("homeRegion", *p.HomeRegion)
	}
	if p.HomeCity != nil {
		add("homeCity", homeCityDoc{Name: p.HomeCity.Name})
	}
	if p.HomeCurrency != nil {
		add("homeCurrency", *p.HomeCurrency)
	}
	if p.Units != nil {
		add("units", unitsToDoc(*p.Units))
	}
	if p.Language != nil {
		add("language", *p.Language)
	}
	if p.FormatLocale != nil {
		add("formatLocale", *p.FormatLocale)
	}
	if p.RegionSource != nil {
		add("regionSource", *p.RegionSource)
	}
	if p.PassportCountry != nil {
		add("passportCountry", *p.PassportCountry)
	}
	if p.TravelStyle != nil {
		add("travelStyle", styleToDoc(*p.TravelStyle))
	}
	if p.PaymentHandles != nil {
		add("paymentHandles", handlesToDoc(*p.PaymentHandles))
	}
	if p.Notifications != nil {
		add("notifications", *p.Notifications)
	}
	if p.OnboardedAt != nil {
		add("onboardedAt", *p.OnboardedAt)
	}
	if _, err := s.doc(id).Update(ctx, ups); err != nil {
		return model.User{}, mapErr("update user", err)
	}
	return s.Get(ctx, id)
}
