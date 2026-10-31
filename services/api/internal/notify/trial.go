package notify

import (
	"context"
	"log/slog"
	"time"
)

// KindTrialEnding is the push sent shortly before a free trial ends.
const KindTrialEnding Kind = "trialEnding"

// TrialLead is how long before the end the reminder goes out.
const TrialLead = 48 * time.Hour

// TrialEnding is a person whose free trial is about to end.
type TrialEnding struct {
	UID    string
	EndsAt time.Time
}

// TrialSource finds trials that are about to end.
type TrialSource interface {
	// EndingBetween returns trials ending in [from, to) for people with no paid
	// Plus and no reminder yet for that end date.
	EndingBetween(ctx context.Context, from, to time.Time) ([]TrialEnding, error)
	// Claim records the reminder for this end date. It is true only for the
	// first caller, so two runs never send the same reminder twice.
	Claim(ctx context.Context, uid string, endsAt time.Time) (bool, error)
}

// TrialReminders sends the "your trial ends soon" push. A scheduler calls Run
// every few hours; the claim makes repeated runs harmless.
type TrialReminders struct {
	Trials TrialSource
	Prefs  PrefSource
	Push   *Service
	Logger *slog.Logger
	Now    func() time.Time
}

// Run sends every reminder that is due and returns how many people were reached.
func (r *TrialReminders) Run(ctx context.Context) (int, error) {
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	from := now()
	due, err := r.Trials.EndingBetween(ctx, from, from.Add(TrialLead))
	if err != nil {
		return 0, err
	}
	reached := 0
	var first error
	for _, t := range due {
		prefs, err := r.Prefs.Prefs(ctx, t.UID)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		if !wants(prefs, KindTrialEnding) {
			continue
		}
		ok, err := r.Trials.Claim(ctx, t.UID, t.EndsAt)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		if !ok {
			continue
		}
		n, err := r.Push.SendToUser(ctx, t.UID, RenderTrialEnding())
		if err != nil && first == nil {
			first = err
		}
		if n > 0 {
			reached++
		}
	}
	return reached, first
}

// RenderTrialEnding writes the reminder. It names no date or price, because
// those come from the store and the app shows them on the plans screen.
func RenderTrialEnding() Message {
	return Message{
		Title: "Your free trial ends soon",
		Body:  "Your trial of Plus ends in 2 days. Tap to see what you keep and how to continue.",
		Data:  map[string]string{"kind": string(KindTrialEnding), "route": "/plans"},
	}
}
