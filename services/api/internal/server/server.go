// Package server wires the HTTP router and the middleware every route shares.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// NewHandler returns the API's router. Routes and request types come from the
// generated code in internal/api; this file only supplies the behaviour.
func NewHandler(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	api.HandlerFromMux(api.NewStrictHandler(handlers{}, nil), mux)
	return logRequests(logger, mux)
}

// handlers implements the generated api.StrictServerInterface.
type handlers struct{}

// GetHealthz reports that the process is up. It checks no dependencies.
func (handlers) GetHealthz(context.Context, api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

// GetMe is a placeholder until bearer-token auth exists (VOY-93). With no
// verifier, nobody can be signed in yet.
func (handlers) GetMe(context.Context, api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	return api.GetMe401JSONResponse{UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{
		Code:    "unauthenticated",
		Message: "Sign-in is not available yet.",
	}}, nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

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
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"durationMs", time.Since(start).Milliseconds(),
		)
	})
}
