package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/notify"
)

// Publisher takes crew events. notify.Crew is the real one.
type Publisher interface {
	Publish(ctx context.Context, ev notify.Event)
}

// WithCrewEvents turns on pushes to the crew when someone joins or adds
// things. Without it nothing is sent.
func WithCrewEvents(p Publisher) Option { return func(h *handlers) { h.events = p } }

// emit reports a crew event. It never fails the request.
func (h *handlers) emit(ctx context.Context, kind notify.Kind, tripID, actor string) {
	if h.events != nil {
		h.events.Publish(ctx, notify.Event{Kind: kind, TripID: tripID, ActorUID: actor})
	}
}
