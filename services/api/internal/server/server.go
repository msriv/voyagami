// Package server wires the HTTP router and the middleware every route shares.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/ask"
	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/billing"
	"github.com/msriv/voyagami/services/api/internal/devlog"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/export"
	"github.com/msriv/voyagami/services/api/internal/importer"
	"github.com/msriv/voyagami/services/api/internal/places"
	"github.com/msriv/voyagami/services/api/internal/rates"
	"github.com/msriv/voyagami/services/api/internal/repo"
	"github.com/msriv/voyagami/services/api/internal/stays"
)

// NewHandler returns the API's router. Routes and request types come from the
// generated code in internal/api; this file only supplies the behaviour. The
// verifier decides who counts as signed in.
func NewHandler(logger *slog.Logger, verifier auth.Verifier, opts ...Option) http.Handler {
	h := &handlers{}
	for _, o := range opts {
		o(h)
	}
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			writeError(w, r, http.StatusBadRequest, codeBadRequest, "The request could not be read.")
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("handler failed", "requestId", requestIDFrom(r.Context()), "error", err)
			writeError(w, r, http.StatusInternalServerError, codeInternal, "Something went wrong.")
		},
	})
	mux := http.NewServeMux()
	if h.devLogs != nil {
		h.mountDevLogs(mux)
	}
	if h.billing != nil && h.billingSecret != "" {
		h.mountBilling(mux)
	}
	if (h.trialJob != nil || h.deletionJob != nil) && h.jobsSecret != "" {
		h.mountJobs(mux)
	}
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{
		BaseRouter:  mux,
		Middlewares: []api.MiddlewareFunc{requireAuth(logger, verifier)},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			writeError(w, r, http.StatusBadRequest, codeBadRequest, "The request could not be read.")
		},
	})
	return withRequestID(logRequests(logger, recoverPanics(logger, jsonMuxErrors(mux))))
}

// handlers implements the generated api.StrictServerInterface.
type handlers struct {
	signIn        SignIn
	profile       Profile
	trips         TripStore
	stops         StopStore
	tasks         TaskStore
	expenses      ExpenseStore
	settlements   SettlementStore
	bookings      BookingStore
	journal       JournalStore
	invites       repo.Invites
	polls         repo.Polls
	meetPoints    repo.MeetPoints
	comments      repo.Comments
	importer      importer.Parser
	devLogs       devlog.Store
	usage         repo.TripUsage
	devices       repo.Devices
	consents      repo.Consents
	productEvents repo.Events
	export        *export.Sources
	support       repo.Support
	supportLimit  *userLimiter
	events        Publisher
	stamps        repo.Stamps
	stampReward   StampRewarder
	billing       billing.Store
	billingSecret string
	trialJob      TrialReminderJob
	deletionJob   TrialReminderJob
	deletions     repo.Deletions
	stayPartners  []stays.Partner
	stayClicks    StayClicks
	staysOn       bool
	catalog       repo.PlaceCatalog
	asker         ask.Drafter
	askCount      AskCounter
	jobsSecret    string
	feed          repo.Feed
	presence      repo.Presence
	rates         rates.Provider
	places        places.Searcher

	plans      *entitlements.Service
	planSource PlanSource
}

// GetHealth reports that the process is up. It checks no dependencies.
func (*handlers) GetHealth(context.Context, api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse{Status: "ok"}, nil
}

// GetMe returns the signed-in person's profile, or just what the token says
// when profiles are off or the profile is not there yet.
func (h *handlers) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	id, ok := auth.FromContext(ctx)
	if !ok {
		// requireAuth guards this route, so this means the wiring is broken.
		return nil, errors.New("GetMe reached without an identity")
	}
	if h.profile != nil {
		user, err := h.profile.Get(ctx, id.UserID)
		if err == nil {
			return api.GetMe200JSONResponse(meFrom(user)), nil
		}
		if !errors.Is(err, repo.ErrNotFound) {
			return nil, err
		}
	}
	me := api.GetMe200JSONResponse{Id: id.UserID}
	if id.DisplayName != "" {
		me.DisplayName = &id.DisplayName
	}
	return me, nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

// Unwrap lets http.ResponseController reach the real writer to flush streams.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// logRequests logs one line per request: method, path, status and duration.
// The query string, headers and bodies are never logged, because they can hold
// personal data.
func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Info("request",
			"requestId", requestIDFrom(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"durationMs", time.Since(start).Milliseconds(),
		)
	})
}
