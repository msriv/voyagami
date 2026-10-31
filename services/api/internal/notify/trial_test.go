package notify

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

type fakeTrials struct {
	due     []TrialEnding
	claimed map[string]bool
	gotTo   time.Time
}

func (f *fakeTrials) EndingBetween(_ context.Context, _, to time.Time) ([]TrialEnding, error) {
	f.gotTo = to
	return f.due, nil
}

func (f *fakeTrials) Claim(_ context.Context, uid string, _ time.Time) (bool, error) {
	if f.claimed[uid] {
		return false, nil
	}
	f.claimed[uid] = true
	return true, nil
}

func TestTrialRemindersRun(t *testing.T) {
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	sender := &recorder{sent: map[string][]Message{}}
	trials := &fakeTrials{claimed: map[string]bool{}, due: []TrialEnding{
		{UID: "a", EndsAt: now.Add(24 * time.Hour)},
		{UID: "b", EndsAt: now.Add(30 * time.Hour)},
		{UID: "c", EndsAt: now.Add(30 * time.Hour)},
	}}
	r := &TrialReminders{
		Trials: trials,
		Prefs:  fakePrefs{"b": {"all": false}},
		Push:   &Service{Devices: &userDevices{}, Sender: sender},
		Logger: slog.New(slog.DiscardHandler),
		Now:    func() time.Time { return now },
	}
	got, err := r.Run(context.Background())
	if err != nil || got != 2 {
		t.Fatalf("first run: reached %d, err %v; want 2", got, err)
	}
	if want := now.Add(TrialLead); !trials.gotTo.Equal(want) {
		t.Errorf("window ends %v, want %v", trials.gotTo, want)
	}
	if len(sender.sent) != 2 || len(sender.sent["b"]) != 0 {
		t.Errorf("sent to %v, want a and c only (b opted out)", sender.sent)
	}
	if m := sender.sent["a"][0]; m.Data["route"] != "/plans" {
		t.Errorf("route %q", m.Data["route"])
	}
	if got, _ := r.Run(context.Background()); got != 0 {
		t.Errorf("second run reached %d, want 0", got)
	}
}
