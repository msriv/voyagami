package server

import (
	"context"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/recapvideo"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithRecapVideo turns on the recap video routes. The renderer is used only to
// turn a stored object into a download link. Without it the routes answer 503.
func WithRecapVideo(store repo.RecapVideos, r recapvideo.Renderer) Option {
	return func(h *handlers) { h.recapVideos, h.recapRender = store, r }
}

// WithRecapJob turns on POST /v1/jobs/recap-videos, which a scheduler calls
// every minute or two to render queued videos. Guarded like the other jobs.
func WithRecapJob(job TrialReminderJob, secret string) Option {
	return func(h *handlers) { h.recapJob, h.jobsSecret = job, secret }
}

const msgNoRecapVideo = "Recap videos are not available right now."

func (h *handlers) recapVideoOff() bool {
	return h.recapVideos == nil || h.recapRender == nil || h.trips == nil
}

func (h *handlers) toAPIRecapVideo(ctx context.Context, v model.RecapVideo) (api.RecapVideo, error) {
	out := api.RecapVideo{Id: v.ID, State: api.RecapVideoState(v.State), Progress: v.Progress, MaxSeconds: v.MaxSeconds, Watermark: v.Watermark, CreatedAt: v.CreatedAt}
	if v.State == model.RecapReady && v.Object != "" {
		u, err := h.recapRender.URL(ctx, v.Object)
		if err != nil {
			return out, err
		}
		out.Url = &u
	}
	if v.Error != "" {
		e := v.Error
		out.Error = &e
	}
	return out, nil
}

// CreateRecapVideo queues a render, or returns the one already running.
func (h *handlers) CreateRecapVideo(ctx context.Context, req api.CreateRecapVideoRequestObject) (api.CreateRecapVideoResponseObject, error) {
	if h.recapVideoOff() {
		return api.CreateRecapVideo503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoRecapVideo))}, nil
	}
	uid, role, deny, err := h.crewRole(ctx, "CreateRecapVideo", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.CreateRecapVideo404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if !model.Can(role, model.ActionEdit) {
		return api.CreateRecapVideo403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	policy := entitlements.RecapPolicy{MaxSeconds: entitlements.Unlimited}
	if h.plans != nil && h.planSource != nil {
		user, err := h.planSource.User(ctx, uid)
		if err != nil {
			return nil, err
		}
		trip, err := h.planSource.Trip(ctx, req.TripId)
		if err != nil {
			return nil, err
		}
		policy = h.plans.Recap(user, &trip)
	}
	job, _, err := h.recapVideos.Enqueue(ctx, model.RecapVideo{TripID: req.TripId, RequestedBy: uid, MaxSeconds: policy.MaxSeconds,
		Watermark: policy.Watermark, CreatedAt: time.Now()})
	if err != nil {
		return nil, err
	}
	out, err := h.toAPIRecapVideo(ctx, job)
	if err != nil {
		return nil, err
	}
	return api.CreateRecapVideo202JSONResponse(out), nil
}

// GetRecapVideo returns the trip's latest render job.
func (h *handlers) GetRecapVideo(ctx context.Context, req api.GetRecapVideoRequestObject) (api.GetRecapVideoResponseObject, error) {
	if h.recapVideoOff() {
		return api.GetRecapVideo503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoRecapVideo))}, nil
	}
	_, _, deny, err := h.crewRole(ctx, "GetRecapVideo", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.GetRecapVideo404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	job, err := h.recapVideos.Latest(ctx, req.TripId)
	if errors.Is(err, repo.ErrNotFound) {
		return api.GetRecapVideo404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No video has been made for this trip."))}, nil
	} else if err != nil {
		return nil, err
	}
	out, err := h.toAPIRecapVideo(ctx, job)
	if err != nil {
		return nil, err
	}
	return api.GetRecapVideo200JSONResponse(out), nil
}
