package firestorerepo

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const consentsCollection = "consents"

// Consents implements repo.Consents on consents/{uid}.
type Consents struct {
	client *firestore.Client
}

var _ repo.Consents = (*Consents)(nil)

// NewConsents returns the Firestore-backed consent store.
func NewConsents(client *firestore.Client) *Consents { return &Consents{client: client} }

type consentsDoc struct {
	Location      string    `firestore:"location"`
	Notifications string    `firestore:"notifications"`
	Photos        string    `firestore:"photos"`
	Analytics     string    `firestore:"analytics"`
	UpdatedAt     time.Time `firestore:"updatedAt"`
}

func (d consentsDoc) model() model.Consents {
	c := model.UnsetConsents()
	for dst, src := range map[*model.ConsentState]string{&c.Location: d.Location, &c.Notifications: d.Notifications, &c.Photos: d.Photos, &c.Analytics: d.Analytics} {
		if src != "" {
			*dst = model.ConsentState(src)
		}
	}
	c.UpdatedAt = d.UpdatedAt
	return c
}

// Get returns the choices; a person with no document has none made.
func (s *Consents) Get(ctx context.Context, uid string) (model.Consents, error) {
	snap, err := s.client.Collection(consentsCollection).Doc(uid).Get(ctx)
	if isNotFound(err) {
		return model.UnsetConsents(), nil
	}
	if err != nil {
		return model.Consents{}, mapErr("get consents", err)
	}
	var d consentsDoc
	if err := snap.DataTo(&d); err != nil {
		return model.Consents{}, mapErr("decode consents", err)
	}
	return d.model(), nil
}

// Update applies the patch and returns the choices after it.
func (s *Consents) Update(ctx context.Context, uid string, p model.ConsentsPatch, now time.Time) (model.Consents, error) {
	if err := p.Validate(); err != nil {
		return model.Consents{}, err
	}
	ref := s.client.Collection(consentsCollection).Doc(uid)
	var out model.Consents
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		cur := model.UnsetConsents()
		snap, err := tx.Get(ref)
		switch {
		case isNotFound(err):
		case err != nil:
			return err
		default:
			var d consentsDoc
			if err := snap.DataTo(&d); err != nil {
				return err
			}
			cur = d.model()
		}
		out = cur.Apply(p, now.UTC().Truncate(time.Microsecond))
		return tx.Set(ref, consentsDoc{string(out.Location), string(out.Notifications), string(out.Photos), string(out.Analytics), out.UpdatedAt})
	})
	return out, mapErr("update consents", err)
}
