package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const (
	msgNoFeed = "Live updates are not turned on."

	// feedHeartbeat keeps proxies from closing a quiet stream.
	feedHeartbeat = 20 * time.Second
	// feedMaxAge ends the stream before Cloud Run's request timeout does; the
	// app reconnects.
	feedMaxAge = 4 * time.Minute
)

// WithFeed turns on the live change stream. It also needs WithTrips.
func WithFeed(f repo.Feed) Option { return func(h *handlers) { h.feed = f } }

// changeStream is the 200 response. It writes events until the client goes,
// the watch breaks or the stream is old enough to end.
type changeStream struct {
	feed    repo.Feed
	tripID  string
	resync  bool
	maxAge  time.Duration
	pulse   time.Duration
	started func() // test hook: called once the stream is open
}

// StreamTripChanges lets any member listen to what changes on the trip.
func (h *handlers) StreamTripChanges(ctx context.Context, req api.StreamTripChangesRequestObject) (api.StreamTripChangesResponseObject, error) {
	if h.feed == nil || h.trips == nil {
		return api.StreamTripChanges503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoFeed))}, nil
	}
	_, _, deny, err := h.crewRole(ctx, "StreamTripChanges", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.StreamTripChanges404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	return streamResponse{ctx: ctx, s: changeStream{
		feed: h.feed, tripID: req.TripId, resync: req.Params.LastEventID != nil && *req.Params.LastEventID != "",
		maxAge: feedMaxAge, pulse: feedHeartbeat,
	}}, nil
}

// streamResponse carries the request context, which the generated visitor
// does not pass on.
type streamResponse struct {
	ctx context.Context
	s   changeStream
}

func (r streamResponse) VisitStreamTripChangesResponse(w http.ResponseWriter) error {
	s := r.s
	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ctx, cancel := context.WithTimeout(r.ctx, s.maxAge)
	defer cancel()

	changes := make(chan model.Change, 64)
	done := make(chan error, 1)
	go func() {
		done <- s.feed.Watch(ctx, s.tripID, func(c model.Change) {
			select {
			case changes <- c:
			case <-ctx.Done():
			}
		})
	}()

	write := func(format string, a ...any) bool {
		if _, err := fmt.Fprintf(w, format, a...); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	if !write(": open\n\n") {
		cancel()
		return nil
	}
	n := 0
	if s.resync {
		n++
		write("id: %d\nevent: resync\ndata: {}\n\n", n)
	}
	if s.started != nil {
		s.started()
	}
	tick := time.NewTicker(s.pulse)
	defer tick.Stop()
	for {
		select {
		case c := <-changes:
			b, _ := json.Marshal(c)
			n++
			if !write("id: %d\nevent: change\ndata: %s\n\n", n, b) {
				cancel()
				return nil
			}
		case <-tick.C:
			if !write(": keep-alive\n\n") {
				cancel()
				return nil
			}
		case <-ctx.Done():
			<-done
			return nil
		case <-done:
			// The watch broke; closing the stream makes the app reconnect.
			return nil
		}
	}
}
