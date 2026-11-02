package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/msriv/voyagami/services/api/internal/analytics"
	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
)

// WithEvents turns on product events. They are kept only for people whose
// analytics consent is granted, so it needs the consent store too.
func WithEvents(s analytics.Sink) Option { return func(h *handlers) { h.productEvents = s } }

const maxEventBatch = 50

const msgNoEvents = "Events are not available right now."

// SendEvents checks a batch of product events and passes the good ones on.
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
	if !h.analyticsOn(ctx, id.UserID) {
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
		c := h.personContext(ctx, id.UserID)
		if b := req.Body.Context; b != nil {
			c.SessionID, c.Platform, c.OSVersion, c.AppVersion, c.AppLanguage =
				str(b.SessionId), str(b.Platform), str(b.OsVersion), str(b.AppVersion), str(b.AppLanguage)
		}
		if err := h.productEvents.Add(ctx, analytics.Records(kept, id.UserID, c, analytics.SourceApp, now)); err != nil {
			return nil, err
		}
	}
	return api.SendEvents200JSONResponse{Accepted: len(kept), Dropped: dropped}, nil
}

// analyticsOn says whether the person turned on "Help improve the app".
func (h *handlers) analyticsOn(ctx context.Context, uid string) bool {
	if h.consents == nil || uid == "" {
		return false
	}
	c, err := h.consents.Get(ctx, uid)
	return err == nil && c.Analytics == model.ConsentGranted
}

// personContext is what the server knows: plan and home country. Best effort.
func (h *handlers) personContext(ctx context.Context, uid string) analytics.Context {
	var c analytics.Context
	if h.profile != nil {
		if u, err := h.profile.Get(ctx, uid); err == nil {
			c.HomeRegion = u.HomeRegion
		}
	}
	if h.plans != nil && h.planSource != nil {
		if u, err := h.planSource.User(ctx, uid); err == nil {
			c.Plan = string(h.entitlementsFor(u, nil).Plan)
		}
	}
	return c
}

// serverEvent records something the server saw (a purchase, a limit). With
// the person's consent it carries their id and the properties; without it,
// the event is counted with neither. Failures are logged, never returned: the
// action it describes already happened.
func (h *handlers) serverEvent(ctx context.Context, uid, name string, props map[string]any) {
	if h.productEvents == nil {
		return
	}
	now := time.Now()
	kept, _ := analytics.Clean([]analytics.Raw{{Name: name, At: now, Props: props}}, now)
	if len(kept) == 0 {
		return
	}
	var c analytics.Context
	if h.analyticsOn(ctx, uid) {
		c = h.personContext(ctx, uid)
	} else {
		uid = ""
	}
	if err := h.productEvents.Add(ctx, analytics.Records(kept, uid, c, analytics.SourceServer, now)); err != nil {
		slog.Default().Error("analytics event failed", "requestId", requestIDFrom(ctx), "event", name, "error", err)
	}
}

// limitReached records that the entitlements service said no.
func (h *handlers) limitReached(ctx context.Context, uid string, f entitlements.Feature, plan api.EntitlementsPlan) {
	h.serverEvent(ctx, uid, "limit_reached", map[string]any{"limit": string(f), "plan": string(plan)})
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
