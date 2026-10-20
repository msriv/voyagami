package server

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/auth"
)

// publicRoutes are the routes that need no token, as the router reports them
// in Request.Pattern. A test checks this list against `security: []` in
// api/openapi.yaml, so the spec stays the source of truth.
var publicRoutes = map[string]bool{
	"GET /healthz": true,
}

// requireAuth checks the bearer token on every route not listed in
// publicRoutes and stores the Identity in the request context.
func requireAuth(logger *slog.Logger, v auth.Verifier) api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if publicRoutes[r.Pattern] {
				next.ServeHTTP(w, r)
				return
			}
			token, ok := bearerToken(r)
			if !ok {
				unauthenticated(w, r, "Send an Authorization: Bearer token.")
				return
			}
			id, err := v.Verify(r.Context(), token)
			switch {
			case errors.Is(err, auth.ErrInvalidToken):
				unauthenticated(w, r, "The token is missing, invalid or expired.")
			case err != nil:
				logger.Error("token verification failed", "requestId", requestIDFrom(r.Context()), "error", err)
				writeError(w, r, http.StatusServiceUnavailable, codeUnavailable, "Sign-in checks are unavailable. Try again.")
			default:
				next.ServeHTTP(w, r.WithContext(auth.NewContext(r.Context(), id)))
			}
		})
	}
}

func unauthenticated(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeError(w, r, http.StatusUnauthorized, codeUnauthenticated, message)
}

// bearerToken extracts the token from "Authorization: Bearer <token>".
func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}
