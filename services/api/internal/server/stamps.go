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

// StampRewarder gives the reward for earning all five stamps.
type StampRewarder interface {
	// GrantStampWeek adds a week of Plus once per account. It reports whether this call granted it.
	GrantStampWeek(ctx context.Context, uid string, now time.Time) (bool, error)
}

// WithStampReward turns on the extra week of Plus for all five stamps. It needs WithStamps.
func WithStampReward(r StampRewarder) Option { return func(h *handlers) { h.stampReward = r } }

// award gives a starter stamp. It never fails the request that earned it: a
// stamp is a nicety, so an error is logged and the stamp is missed.
func (h *handlers) award(ctx context.Context, uid string, kind model.StampKind) {
	if h.stamps == nil {
		return
	}
	won, err := h.stamps.Award(ctx, uid, kind, time.Now())
	if err != nil {
		slog.Default().Warn("award stamp failed", "requestId", requestIDFrom(ctx), "stamp", kind, "error", err)
		return
	}
	if won {
		h.rewardIfComplete(ctx, uid)
	}
}

// countPlace counts a place the person added toward the Add 3 places stamp.
func (h *handlers) countPlace(ctx context.Context, uid string) {
	if h.stamps == nil {
		return
	}
	won, err := h.stamps.AddPlace(ctx, uid, time.Now())
	if err != nil {
		slog.Default().Warn("count place failed", "requestId", requestIDFrom(ctx), "error", err)
		return
	}
	if won {
		h.rewardIfComplete(ctx, uid)
	}
}

// rewardIfComplete grants the Plus week when the stamp just earned was the fifth.
func (h *handlers) rewardIfComplete(ctx context.Context, uid string) {
	if h.stampReward == nil {
		return
	}
	earned, err := h.stamps.List(ctx, uid)
	if err != nil || len(earned) < len(model.StampKinds) {
		if err != nil {
			slog.Default().Warn("check stamps failed", "requestId", requestIDFrom(ctx), "error", err)
		}
		return
	}
	if _, err := h.stampReward.GrantStampWeek(ctx, uid, time.Now()); err != nil {
		slog.Default().Warn("grant stamp week failed", "requestId", requestIDFrom(ctx), "error", err)
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
