package server

import (
	"context"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithDeletions turns on account deletion requests. Without it the routes answer 503.
func WithDeletions(d repo.Deletions) Option { return func(h *handlers) { h.deletions = d } }

const (
	msgNoDeletions         = "Account deletion is not available right now."
	codeSubscriptionActive = "subscription_active"
	msgSubscriptionActive  = "Cancel your subscription in the App Store or Google Play first, then delete your account."
)

func toAPIDeletion(d model.Deletion) api.Deletion {
	return api.Deletion{Scheduled: true, DeleteOn: &d.DeleteOn}
}

// ScheduleMyDeletion asks to delete the account in 30 days. A subscription that
// still renews blocks it, so the person is not charged for an account they no
// longer have.
func (h *handlers) ScheduleMyDeletion(ctx context.Context, _ api.ScheduleMyDeletionRequestObject) (api.ScheduleMyDeletionResponseObject, error) {
	id, err := identity(ctx, "ScheduleMyDeletion")
	if err != nil {
		return nil, err
	}
	if h.deletions == nil {
		return api.ScheduleMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoDeletions))}, nil
	}
	if h.plans != nil && h.planSource != nil {
		user, err := h.planSource.User(ctx, id.UserID)
		if err != nil {
			return nil, err
		}
		if user.Sub != nil && user.Sub.WillRenew {
			return api.ScheduleMyDeletion409JSONResponse{ConflictJSONResponse: api.ConflictJSONResponse(errBody(ctx, codeSubscriptionActive, msgSubscriptionActive))}, nil
		}
	}
	d, err := h.deletions.Schedule(ctx, id.UserID, time.Now())
	if err != nil {
		return nil, err
	}
	return api.ScheduleMyDeletion200JSONResponse(toAPIDeletion(d)), nil
}

// GetMyDeletion says whether deletion is scheduled.
func (h *handlers) GetMyDeletion(ctx context.Context, _ api.GetMyDeletionRequestObject) (api.GetMyDeletionResponseObject, error) {
	id, err := identity(ctx, "GetMyDeletion")
	if err != nil {
		return nil, err
	}
	if h.deletions == nil {
		return api.GetMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoDeletions))}, nil
	}
	d, err := h.deletions.Get(ctx, id.UserID)
	if errors.Is(err, repo.ErrNotFound) {
		return api.GetMyDeletion200JSONResponse(api.Deletion{}), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetMyDeletion200JSONResponse(toAPIDeletion(d)), nil
}

// CancelMyDeletion keeps the account.
func (h *handlers) CancelMyDeletion(ctx context.Context, _ api.CancelMyDeletionRequestObject) (api.CancelMyDeletionResponseObject, error) {
	id, err := identity(ctx, "CancelMyDeletion")
	if err != nil {
		return nil, err
	}
	if h.deletions == nil {
		return api.CancelMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoDeletions))}, nil
	}
	if err := h.deletions.Cancel(ctx, id.UserID); err != nil {
		return nil, err
	}
	return api.CancelMyDeletion204Response{}, nil
}
