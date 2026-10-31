package firestorerepo

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/billing"
)

const (
	billingEventsCollection = "billingEvents"
	tripPassesCollection    = "tripPasses"
)

// Billing writes RevenueCat events into entitlements/{uid} and tripPasses/{transactionId}.
type Billing struct {
	client *firestore.Client
}

var _ billing.Store = (*Billing)(nil)

// NewBilling returns the Firestore-backed billing store.
func NewBilling(client *firestore.Client) *Billing { return &Billing{client: client} }

// Apply records one event. An event ID seen before is skipped.
func (s *Billing) Apply(ctx context.Context, e billing.Event) error {
	if e.Anonymous() {
		return nil
	}
	seen := s.client.Collection(billingEventsCollection).Doc(e.ID)
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		if snap, err := tx.Get(seen); err == nil && snap.Exists() {
			return nil
		} else if err != nil && !isNotFound(err) {
			return err
		}
		if e.IsTripPass() {
			return s.applyPass(tx, e, seen)
		}
		return s.applyPlus(tx, e, seen)
	})
	return mapErr("apply billing event", err)
}

func (s *Billing) applyPlus(tx *firestore.Transaction, e billing.Event, seen *firestore.DocumentRef) error {
	ref := s.client.Collection(entitlementsCollection).Doc(e.UID)
	var d entitlementsDoc
	snap, err := tx.Get(ref)
	switch {
	case isNotFound(err):
	case err != nil:
		return err
	default:
		if err := snap.DataTo(&d); err != nil {
			return err
		}
	}
	cur := billing.Plus{
		Active: d.Plus.Active, ProductID: d.Plus.ProductID, ExpiresAt: d.Plus.ExpiresAt, WillRenew: d.Plus.WillRenew,
		BillingIssue: d.Plus.BillingIssue, LastEventAt: d.Billing.LastEventAt,
	}
	if d.Trial != nil {
		cur.TrialEndsAt = d.Trial.EndsAt
	}
	next, changed := billing.Next(cur, e)
	if !changed {
		return tx.Set(seen, map[string]any{"type": e.Type, "at": e.At, "ignored": true})
	}
	trial := map[string]any{"endsAt": firestore.Delete}
	if !next.TrialEndsAt.IsZero() {
		trial["endsAt"] = next.TrialEndsAt
	}
	expires := any(firestore.Delete)
	if !next.ExpiresAt.IsZero() {
		expires = next.ExpiresAt
	}
	if err := tx.Set(ref, map[string]any{
		"plus": map[string]any{
			"active": next.Active, "productId": next.ProductID, "expiresAt": expires,
			"willRenew": next.WillRenew, "billingIssue": next.BillingIssue,
		},
		"trial":   trial,
		"billing": map[string]any{"lastEventAt": next.LastEventAt},
	}, firestore.MergeAll); err != nil {
		return err
	}
	return tx.Set(seen, map[string]any{"type": e.Type, "at": e.At})
}

func (s *Billing) applyPass(tx *firestore.Transaction, e billing.Event, seen *firestore.DocumentRef) error {
	id := e.TransactionID
	if id == "" {
		id = e.ID
	}
	ref := s.client.Collection(tripPassesCollection).Doc(id)
	existing, err := tx.Get(ref)
	if err != nil && !isNotFound(err) {
		return err
	}
	switch {
	case e.Refunded():
		if err := tx.Set(ref, map[string]any{"refundedAt": e.At}, firestore.MergeAll); err != nil {
			return err
		}
	case e.Type == "NON_RENEWING_PURCHASE" && e.TripID != "":
		doc := map[string]any{"tripId": e.TripID, "purchasedBy": e.UID, "productId": e.ProductID, "purchasedAt": e.At}
		if err != nil || !existing.Exists() {
			doc["refundedAt"] = nil // a null field is what HasPass looks for
		} // else a refund that arrived first keeps its refundedAt
		if err := tx.Set(ref, doc, firestore.MergeAll); err != nil {
			return err
		}
	}
	return tx.Set(seen, map[string]any{"type": e.Type, "at": e.At})
}

// HasPass reports whether the trip has a Trip Pass that has not been refunded.
func (s *Billing) HasPass(ctx context.Context, tripID string) (bool, error) {
	docs, err := s.client.Collection(tripPassesCollection).
		Where("tripId", "==", tripID).Where("refundedAt", "==", nil).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return false, mapErr("find trip pass", err)
	}
	return len(docs) > 0, nil
}
