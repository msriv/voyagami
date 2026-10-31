package firestorerepo

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/notify"
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
		Active       bool      `firestore:"active"`
		ProductID    string    `firestore:"productId"`
		ExpiresAt    time.Time `firestore:"expiresAt"`
		WillRenew    bool      `firestore:"willRenew"`
		BillingIssue bool      `firestore:"billingIssue"`
	} `firestore:"plus"`
	Billing struct {
		LastEventAt time.Time `firestore:"lastEventAt"`
	} `firestore:"billing"`
	Trial *struct {
		StartedAt time.Time `firestore:"startedAt"`
		EndsAt    time.Time `firestore:"endsAt"`
		// ReminderFor is the end date the "trial ends soon" push was sent for.
		ReminderFor *time.Time `firestore:"reminderFor"`
	} `firestore:"trial"`
	Usage accountUsage `firestore:"usage"`
	// StampWeekGrantedAt is set once the extra week for all five stamps was given.
	Rewards struct {
		StampWeekGrantedAt *time.Time `firestore:"stampWeekGrantedAt"`
	} `firestore:"rewards"`
}

type accountUsage struct {
	// PeriodStart is the first instant (UTC) of the month the counters belong to.
	PeriodStart time.Time `firestore:"periodStart"`
	AskDrafts   int       `firestore:"askDrafts"`
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
	// A subscription whose end date has passed is over, even if the expiry event never arrived.
	subscribed := d.Plus.Active && (d.Plus.ExpiresAt.IsZero() || d.Plus.ExpiresAt.After(time.Now()))
	return subscribed || trialing, d.Usage.askDraftsIn(time.Now()), nil
}

// monthStart is the first instant of t's month in UTC. Counters reset on it.
func monthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// askDraftsIn is the Ask draft count for the month that holds now. A counter
// from an earlier month reads as zero, so nothing has to run at midnight.
func (u accountUsage) askDraftsIn(now time.Time) int {
	if !u.PeriodStart.Equal(monthStart(now)) {
		return 0
	}
	return u.AskDrafts
}

// AddAskDraft counts one Ask draft for the person and returns this month's
// total. The first draft of a new month starts the count again.
func (s *Entitlements) AddAskDraft(ctx context.Context, uid string, now time.Time) (int, error) {
	ref := s.client.Collection(entitlementsCollection).Doc(uid)
	var total int
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
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
		total = d.Usage.askDraftsIn(now) + 1
		return tx.Set(ref, map[string]any{
			"usage": map[string]any{"periodStart": monthStart(now), "askDrafts": total},
		}, firestore.MergeAll)
	})
	return total, mapErr("add ask draft", err)
}

// StampWeek is the Plus time given for earning all five starter stamps.
const StampWeek = 7 * 24 * time.Hour

// GrantStampWeek adds a week of Plus, once per account. A running trial is
// extended by a week; an ended or missing one becomes a week from now. Someone
// with a paid subscription gets nothing, and the grant is not kept for later.
func (s *Entitlements) GrantStampWeek(ctx context.Context, uid string, now time.Time) (bool, error) {
	ref := s.client.Collection(entitlementsCollection).Doc(uid)
	granted := false
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		granted = false
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
		if d.Rewards.StampWeekGrantedAt != nil {
			return nil
		}
		if d.paid(now) {
			return nil
		}
		started, from := now, now
		if d.Trial != nil && !d.Trial.StartedAt.IsZero() {
			started = d.Trial.StartedAt
		}
		if d.Trial != nil && d.Trial.EndsAt.After(now) {
			from = d.Trial.EndsAt
		}
		granted = true
		return tx.Set(ref, map[string]any{
			"trial":   map[string]any{"startedAt": started.UTC(), "endsAt": from.Add(StampWeek).UTC()},
			"rewards": map[string]any{"stampWeekGrantedAt": now.UTC()},
		}, firestore.MergeAll)
	})
	return granted && err == nil, mapErr("grant stamp week", err)
}

// EndingBetween lists trials that end in [from, to) for people without a paid
// subscription and not yet reminded about that end date.
func (s *Entitlements) EndingBetween(ctx context.Context, from, to time.Time) ([]notify.TrialEnding, error) {
	it := s.client.Collection(entitlementsCollection).
		Where("trial.endsAt", ">=", from).Where("trial.endsAt", "<", to).Documents(ctx)
	defer it.Stop()
	var out []notify.TrialEnding
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, mapErr("list ending trials", err)
		}
		var d entitlementsDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode entitlements", err)
		}
		if d.Trial == nil || d.paid(from) || d.Trial.ReminderFor != nil && d.Trial.ReminderFor.Equal(d.Trial.EndsAt) {
			continue
		}
		out = append(out, notify.TrialEnding{UID: snap.Ref.ID, EndsAt: d.Trial.EndsAt})
	}
}

// Claim records that the reminder for this end date is being sent. Only the
// first caller gets true; a trial that was extended since has a new end date
// and can be reminded again.
func (s *Entitlements) Claim(ctx context.Context, uid string, endsAt time.Time) (bool, error) {
	ref := s.client.Collection(entitlementsCollection).Doc(uid)
	claimed := false
	err := s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		claimed = false
		snap, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var d entitlementsDoc
		if err := snap.DataTo(&d); err != nil {
			return err
		}
		if d.Trial == nil || !d.Trial.EndsAt.Equal(endsAt) || d.Trial.ReminderFor != nil && d.Trial.ReminderFor.Equal(endsAt) {
			return nil
		}
		claimed = true
		return tx.Set(ref, map[string]any{"trial": map[string]any{"reminderFor": endsAt.UTC()}}, firestore.MergeAll)
	})
	return claimed && err == nil, mapErr("claim trial reminder", err)
}

// paid is true when a paid subscription is running at t.
func (d entitlementsDoc) paid(t time.Time) bool {
	return d.Plus.Active && (d.Plus.ExpiresAt.IsZero() || d.Plus.ExpiresAt.After(t))
}
