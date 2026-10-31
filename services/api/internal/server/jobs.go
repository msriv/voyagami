package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
)

// TrialReminderJob sends the trial-ending pushes that are due.
type TrialReminderJob interface {
	Run(ctx context.Context) (int, error)
}

// WithJobs turns on POST /v1/jobs/trial-reminders, which a scheduler calls every
// few hours. It sits outside the OpenAPI contract and the sign-in check and
// takes a shared secret as `Authorization: Bearer <secret>`. An empty secret
// leaves the route off.
func WithJobs(trial TrialReminderJob, secret string) Option {
	return func(h *handlers) { h.trialJob, h.jobsSecret = trial, secret }
}

func (h *handlers) mountJobs(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/jobs/trial-reminders", func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, []byte("Bearer "+h.jobsSecret)) != 1 {
			writeError(w, r, http.StatusUnauthorized, codeUnauthenticated, "Sign-in is required.")
			return
		}
		n, err := h.trialJob.Run(r.Context())
		if err != nil {
			// Reminders already claimed went out; the next run picks up the rest.
			slog.Default().Error("trial reminders failed", "requestId", requestIDFrom(r.Context()), "reached", n, "error", err)
			writeError(w, r, http.StatusInternalServerError, codeInternal, "Something went wrong.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"reached": n})
	})
}
