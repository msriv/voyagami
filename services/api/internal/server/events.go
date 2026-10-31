package server

import (
	"context"
	"time"

	"github.com/msriv/voyagami/services/api/internal/analytics"
	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithEvents turns on product events. They are kept only for people whose
// analytics consent is granted, so it needs the consent store too.
func WithEvents(e repo.Events) Option { return func(h *handlers) { h.productEvents = e } }

const maxEventBatch = 50

const msgNoEvents = "Events are not available right now."

// SendEvents checks a batch of product events and stores the good ones.
func (h *handlers) SendEvents(ctx context.Context, req api.SendEventsRequestObject) (api.SendEventsResponseObject, error) {
	id, err := identity(ctx, "SendEvents")
	if err != nil {
		return nil, err
	}
	if h.productEvents == nil || h.consents == nil {
		return api.SendEvents503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoEvents))}, nil
	}
	in := req.Body.Events
	if len(in) == 0 || len(in) > maxEventBatch {
		return api.SendEvents400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "send 1 to 50 events at a time")}, nil
	}
	c, err := h.consents.Get(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	if c.Analytics != model.ConsentGranted {
		return api.SendEvents200JSONResponse{Accepted: 0, Dropped: len(in)}, nil
	}
	raw := make([]analytics.Raw, 0, len(in))
	for _, e := range in {
		r := analytics.Raw{Name: e.Name, At: e.At}
		if e.Props != nil {
			r.Props = *e.Props
		}
		raw = append(raw, r)
	}
	now := time.Now()
	kept, dropped := analytics.Clean(raw, now)
	if len(kept) > 0 {
		if err := h.productEvents.Add(ctx, id.UserID, kept, now); err != nil {
			return nil, err
		}
	}
	return api.SendEvents200JSONResponse{Accepted: len(kept), Dropped: dropped}, nil
}
