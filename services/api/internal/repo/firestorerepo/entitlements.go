package firestorerepo

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

const entitlementsCollection = "entitlements"

// Entitlements reads the server-owned entitlements/{uid} documents.
type Entitlements struct {
	client *firestore.Client
}

// NewEntitlements returns the Firestore-backed entitlements reader.
func NewEntitlements(client *firestore.Client) *Entitlements { return &Entitlements{client: client} }

type entitlementsDoc struct {
	Plus struct {
		Active bool `firestore:"active"`
	} `firestore:"plus"`
	Trial *struct {
		EndsAt time.Time `firestore:"endsAt"`
	} `firestore:"trial"`
	Usage struct {
		AskDrafts int `firestore:"askDrafts"`
	} `firestore:"usage"`
}

// Account returns whether the person has Plus right now (a subscription or a
// running trial) and their Ask drafts this month. A person with no document
// is on Free.
func (s *Entitlements) Account(ctx context.Context, uid string) (plus bool, askDrafts int, err error) {
	snap, err := s.client.Collection(entitlementsCollection).Doc(uid).Get(ctx)
	if isNotFound(err) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, mapErr("get entitlements", err)
	}
	var d entitlementsDoc
	if err := snap.DataTo(&d); err != nil {
		return false, 0, mapErr("decode entitlements", err)
	}
	trialing := d.Trial != nil && d.Trial.EndsAt.After(time.Now())
	return d.Plus.Active || trialing, d.Usage.AskDrafts, nil
}
