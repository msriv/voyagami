package server

import (
	"encoding/json"
	"net/http"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// Stable error codes. The app switches on these; the message is for developers.
const (
	codeBadRequest       = "bad_request"
	codeUnauthenticated  = "unauthenticated"
	codeNotFound         = "not_found"
	codeMethodNotAllowed = "method_not_allowed"
	codeInternal         = "internal"
	codeUnavailable      = "unavailable"
)

// writeError sends the API's one error shape, tagged with the request ID.
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	body := api.Error{Code: code, Message: message}
	if id := requestIDFrom(r.Context()); id != "" {
		body.RequestId = &id
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// jsonMuxErrors turns the router's plain-text 404 and 405 replies into the
// standard error JSON. Handlers that set their own Content-Type are untouched.
func jsonMuxErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&muxErrorWriter{ResponseWriter: w, r: r}, r)
	})
}

type muxErrorWriter struct {
	http.ResponseWriter
	r       *http.Request
	swapped bool
}

func (m *muxErrorWriter) WriteHeader(status int) {
	if (status == http.StatusNotFound || status == http.StatusMethodNotAllowed) &&
		m.Header().Get("Content-Type") != "application/json" {
		m.swapped = true
		code, msg := codeNotFound, "No such endpoint."
		if status == http.StatusMethodNotAllowed {
			code, msg = codeMethodNotAllowed, "That method is not allowed here."
		}
		m.Header().Del("X-Content-Type-Options")
		m.Header().Del("Content-Length")
		writeError(m.ResponseWriter, m.r, status, code, msg)
		return
	}
	m.ResponseWriter.WriteHeader(status)
}

func (m *muxErrorWriter) Write(b []byte) (int, error) {
	if m.swapped {
		return len(b), nil // drop the router's text body
	}
	return m.ResponseWriter.Write(b)
}
