// Package billing turns RevenueCat webhook events into entitlement changes.
// It holds no I/O: Parse reads a payload and Next works out the new state, so
// both are tested with recorded payloads. The store applies them.
package billing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Event is the part of a RevenueCat event that the plan needs.
type Event struct {
	ID            string
	Type          string
	UID           string // RevenueCat app_user_id; the app sets it to the account ID
	ProductID     string
	TransactionID string
	PeriodType    string // TRIAL, INTRO or NORMAL
	CancelReason  string
	TripID        string // from the tripId subscriber attribute, Trip Pass only
	At            time.Time
	ExpiresAt     time.Time
}

// Refunded reports a cancellation that ends access at once.
func (e Event) Refunded() bool {
	return e.Type == "CANCELLATION" && e.CancelReason == "CUSTOMER_SUPPORT"
}

// IsTripPass reports whether the product is a Trip Pass.
//
// shortcut: the product ID must contain "trip_pass"; swap for an explicit
// product table when the store products exist (VOY-190).
func (e Event) IsTripPass() bool { return strings.Contains(strings.ToLower(e.ProductID), "trip_pass") }

// Anonymous reports an app_user_id RevenueCat made up before sign-in. There is
// no account to change, so the event is acknowledged and dropped.
func (e Event) Anonymous() bool { return strings.HasPrefix(e.UID, "$RCAnonymousID") || e.UID == "" }

type wire struct {
	Event struct {
		ID               string `json:"id"`
		Type             string `json:"type"`
		AppUserID        string `json:"app_user_id"`
		ProductID        string `json:"product_id"`
		TransactionID    string `json:"transaction_id"`
		PeriodType       string `json:"period_type"`
		CancelReason     string `json:"cancel_reason"`
		EventTimestampMS int64  `json:"event_timestamp_ms"`
		ExpirationAtMS   int64  `json:"expiration_at_ms"`
		SubscriberAttrs  map[string]struct {
			Value string `json:"value"`
		} `json:"subscriber_attributes"`
	} `json:"event"`
}

// Parse reads a webhook body.
func Parse(body []byte) (Event, error) {
	var w wire
	if err := json.Unmarshal(body, &w); err != nil {
		return Event{}, err
	}
	e := w.Event
	if e.ID == "" || e.Type == "" {
		return Event{}, errors.New("event id and type are required")
	}
	out := Event{
		ID: e.ID, Type: e.Type, UID: e.AppUserID, ProductID: e.ProductID, TransactionID: e.TransactionID,
		PeriodType: e.PeriodType, CancelReason: e.CancelReason, TripID: e.SubscriberAttrs["tripId"].Value,
		At: time.UnixMilli(e.EventTimestampMS).UTC(),
	}
	if e.ExpirationAtMS > 0 {
		out.ExpiresAt = time.UnixMilli(e.ExpirationAtMS).UTC()
	}
	return out, nil
}

// Plus is a person's subscription as the server stores it.
type Plus struct {
	Active       bool
	ProductID    string
	ExpiresAt    time.Time
	WillRenew    bool
	BillingIssue bool
	TrialEndsAt  time.Time // zero when there is no trial
	LastEventAt  time.Time // newest event applied; older ones are ignored
}

// Next applies one subscription event to the stored state. It returns the new
// state and whether anything changed. Events older than the newest one applied
// change nothing, so a replay or a late delivery cannot undo a later state.
func Next(cur Plus, e Event) (Plus, bool) {
	if !e.At.After(cur.LastEventAt) {
		return cur, false
	}
	n := cur
	n.LastEventAt = e.At
	switch e.Type {
	case "INITIAL_PURCHASE", "RENEWAL", "PRODUCT_CHANGE", "UNCANCELLATION":
		n.ProductID, n.WillRenew, n.BillingIssue = e.ProductID, true, false
		if e.PeriodType == "TRIAL" {
			n.TrialEndsAt = e.ExpiresAt
		} else {
			n.Active, n.ExpiresAt, n.TrialEndsAt = true, e.ExpiresAt, time.Time{}
		}
	case "CANCELLATION":
		if e.Refunded() {
			n.Active, n.WillRenew, n.TrialEndsAt = false, false, time.Time{}
		} else {
			n.WillRenew = false // access runs to the end of the paid period
		}
	case "EXPIRATION":
		n.Active, n.WillRenew, n.TrialEndsAt = false, false, time.Time{}
	case "BILLING_ISSUE":
		n.BillingIssue = true
	default:
		return cur, false
	}
	return n, true
}

// Store applies events to saved state. Apply is safe to call twice with the
// same event and with events out of order.
type Store interface {
	Apply(ctx context.Context, e Event) error
}
