package server

import (
	"context"
	"strconv"
	"time"

	"crypto/subtle"
	"encoding/json"
	"github.com/msriv/voyagami/services/api/internal/analytics"
	"log/slog"
	"net/http"
)

// TrialReminderJob is work a scheduler triggers: it returns how many things it did.
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

// WithDeletionJob turns on POST /v1/jobs/account-deletions, which a scheduler
// calls daily to purge accounts whose 30 days are over. It is guarded like the
// other jobs; the reply counts the accounts purged.
func WithDeletionJob(job TrialReminderJob, secret string) Option {
	return func(h *handlers) { h.deletionJob, h.jobsSecret = job, secret }
}

func (h *handlers) mountJobs(mux *http.ServeMux) {
	if h.trialJob != nil {
		h.mountJob(mux, "trial-reminders", "reached", "trial reminders failed", h.trialJob)
	}
	if h.deletionJob != nil {
		h.mountJob(mux, "account-deletions", "purged", "account deletions failed", h.deletionJob)
	}
}

func (h *handlers) mountJob(mux *http.ServeMux, name, key, failure string, job TrialReminderJob) {
	mux.HandleFunc("POST /v1/jobs/"+name, func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, []byte("Bearer "+h.jobsSecret)) != 1 {
			writeError(w, r, http.StatusUnauthorized, codeUnauthenticated, "Sign-in is required.")
			return
		}
		n, err := job.Run(r.Context())
		if err != nil {
			// Work already done stays done; the next run picks up the rest.
			slog.Default().Error(failure, "requestId", requestIDFrom(r.Context()), "done", n, "error", err)
			writeError(w, r, http.StatusInternalServerError, codeInternal, "Something went wrong.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{key: n})
	})
}

// EventRows reads stored product events for the weekly review.
type EventRows interface {
	Rows(ctx context.Context, sinceDay string, limit int) ([]analytics.Row, error)
}

// WithAnalyticsReport turns on GET /v1/jobs/analytics-report?days=28, the
// counts for the weekly review. Guarded by the same bearer secret as the jobs;
// there is no in-app screen. Mihir reads it with curl (docs/analytics.md).
func WithAnalyticsReport(src EventRows, secret string) Option {
	return func(h *handlers) { h.eventRows, h.jobsSecret = src, secret }
}

const reportRowLimit = 100000 // shortcut: a scan cap, see firestorerepo Events.Rows

func (h *handlers) mountReport(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/jobs/analytics-report", func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, []byte("Bearer "+h.jobsSecret)) != 1 {
			writeError(w, r, http.StatusUnauthorized, codeUnauthenticated, "Sign-in is required.")
			return
		}
		days := 28
		if v := r.URL.Query().Get("days"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 365 {
				writeError(w, r, http.StatusBadRequest, codeBadRequest, "days must be 1 to 365.")
				return
			}
			days = n
		}
		since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
		rows, err := h.eventRows.Rows(r.Context(), since, reportRowLimit)
		if err != nil {
			slog.Default().Error("analytics report failed", "requestId", requestIDFrom(r.Context()), "error", err)
			writeError(w, r, http.StatusInternalServerError, codeInternal, "Something went wrong.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(analytics.Summarize(since, rows))
	})
}
