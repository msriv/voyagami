package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithStamps turns on the starter stamps. Without it nothing is awarded and
// the stamps route answers 503.
func WithStamps(s repo.Stamps) Option { return func(h *handlers) { h.stamps = s } }

// award gives a starter stamp. It never fails the request that earned it: a
// stamp is a nicety, so an error is logged and the stamp is missed.
func (h *handlers) award(ctx context.Context, uid string, kind model.StampKind) {
	if h.stamps == nil {
		return
	}
	if _, err := h.stamps.Award(ctx, uid, kind, time.Now()); err != nil {
		slog.Default().Warn("award stamp failed", "requestId", requestIDFrom(ctx), "stamp", kind, "error", err)
	}
}

// countPlace counts a place the person added toward the Add 3 places stamp.
func (h *handlers) countPlace(ctx context.Context, uid string) {
	if h.stamps == nil {
		return
	}
	if err := h.stamps.AddPlace(ctx, uid, time.Now()); err != nil {
		slog.Default().Warn("count place failed", "requestId", requestIDFrom(ctx), "error", err)
	}
}

// ListMyStamps lists all five starter stamps with the ones earned marked.
func (h *handlers) ListMyStamps(ctx context.Context, _ api.ListMyStampsRequestObject) (api.ListMyStampsResponseObject, error) {
	id, err := identity(ctx, "ListMyStamps")
	if err != nil {
		return nil, err
	}
	if h.stamps == nil {
		return api.ListMyStamps503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, "Stamps are not available right now."))}, nil
	}
	earned, err := h.stamps.List(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	at := map[model.StampKind]time.Time{}
	for _, e := range earned {
		at[e.Kind] = e.EarnedAt
	}
	out := api.StampList{Stamps: make([]api.Stamp, 0, len(model.StampKinds)), Total: len(model.StampKinds)}
	for _, k := range model.StampKinds {
		s := api.Stamp{Kind: api.StampKind(k)}
		if t, ok := at[k]; ok {
			s.Earned, s.EarnedAt = true, &t
			out.Earned++
		}
		out.Stamps = append(out.Stamps, s)
	}
	return api.ListMyStamps200JSONResponse(out), nil
}
