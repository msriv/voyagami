package main

import (
	"context"
	"log/slog"

	"github.com/msriv/voyagami/services/api/internal/config"
	"github.com/msriv/voyagami/services/api/internal/notify"
	"github.com/msriv/voyagami/services/api/internal/repo"
	"github.com/msriv/voyagami/services/api/internal/repo/firestorerepo"
)

// userPrefs reads a person's notification settings from their profile.
type userPrefs struct{ users repo.Users }

func (p userPrefs) Prefs(ctx context.Context, uid string) (map[string]bool, error) {
	u, err := p.users.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	return u.Notifications, nil
}

// newCrewEvents builds the crew push fan-out. Dev only logs pushes; elsewhere
// they go through FCM.
func newCrewEvents(cfg config.Config, logger *slog.Logger, trips *firestorerepo.Trips, users repo.Users, devices repo.Devices) *notify.Crew {
	var sender notify.Notifier = notify.Log{Logger: logger}
	if cfg.Env != "dev" && cfg.GCPProject != "" {
		sender = &notify.FCM{Project: cfg.GCPProject}
	}
	return &notify.Crew{
		Trips: trips, Prefs: userPrefs{users}, Logger: logger,
		Push: &notify.Service{Devices: devices, Sender: sender, Logger: logger},
	}
}

// newTrialReminders builds the trial-ending push job.
func newTrialReminders(cfg config.Config, logger *slog.Logger, users repo.Users, devices repo.Devices, trials notify.TrialSource) *notify.TrialReminders {
	var sender notify.Notifier = notify.Log{Logger: logger}
	if cfg.Env != "dev" && cfg.GCPProject != "" {
		sender = &notify.FCM{Project: cfg.GCPProject}
	}
	return &notify.TrialReminders{
		Trials: trials, Prefs: userPrefs{users}, Logger: logger,
		Push: &notify.Service{Devices: devices, Sender: sender, Logger: logger},
	}
}
