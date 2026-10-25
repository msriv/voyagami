package server

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/msriv/voyagami/services/api/internal/devlog"
)

// WithDevLogs turns on POST and GET /v1/dev/logs, an open sink for app crash
// reports. Wire it in dev only: the routes take no sign-in because a crashed
// app may not have one. They are outside the OpenAPI contract on purpose.
func WithDevLogs(s devlog.Store) Option { return func(h *handlers) { h.devLogs = s } }

// logLimiter allows each caller a short burst and a steady trickle.
type logLimiter struct {
	mu sync.Mutex
	by map[string]*rate.Limiter
}

func (l *logLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.by == nil || len(l.by) > 10000 { // shortcut: reset when full, fine for one dev project
		l.by = map[string]*rate.Limiter{}
	}
	lim, ok := l.by[key]
	if !ok {
		lim = rate.NewLimiter(rate.Every(2*time.Second), 20)
		l.by[key] = lim
	}
	return lim.Allow()
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Cloud Run appends the real client last.
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *handlers) mountDevLogs(mux *http.ServeMux) {
	lim := &logLimiter{}
	mux.HandleFunc("POST /v1/dev/logs", func(w http.ResponseWriter, r *http.Request) {
		if !lim.allow(clientIP(r)) {
			writeError(w, r, http.StatusTooManyRequests, codeRateLimited, "Too many reports. Slow down.")
			return
		}
		var e devlog.Entry
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, devlog.MaxBodyBytes))
		if err := dec.Decode(&e); err != nil {
			writeError(w, r, http.StatusBadRequest, codeBadRequest, "The report could not be read.")
			return
		}
		if e.DeviceID == "" || e.Message == "" || !devlog.Levels[e.Level] {
			writeError(w, r, http.StatusBadRequest, codeBadRequest, "deviceId, message and a level (debug, info, warn, error, fatal) are required.")
			return
		}
		e = devlog.Clean(e)
		e.CreatedAt = time.Now().UTC()
		if err := h.devLogs.Add(r.Context(), e); err != nil {
			writeError(w, r, http.StatusInternalServerError, codeInternal, "Something went wrong.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /v1/dev/logs", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var since time.Time
		if s := q.Get("since"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				writeError(w, r, http.StatusBadRequest, codeBadRequest, "since must be an RFC 3339 time.")
				return
			}
			since = t
		}
		limit := 50
		if s := q.Get("limit"); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || n < 1 || n > 200 {
				writeError(w, r, http.StatusBadRequest, codeBadRequest, "limit must be 1 to 200.")
				return
			}
			limit = n
		}
		list, err := h.devLogs.List(r.Context(), q.Get("deviceId"), since, limit)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, codeInternal, "Something went wrong.")
			return
		}
		if list == nil {
			list = []devlog.Entry{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": list})
	})
}
