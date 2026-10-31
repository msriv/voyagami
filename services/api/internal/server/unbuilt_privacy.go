package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// Not built yet: answers 503 until its ticket replaces it. Delete this file when the real code lands.

func (h *handlers) CancelMyDeletion(ctx context.Context, _ api.CancelMyDeletionRequestObject) (api.CancelMyDeletionResponseObject, error) {
	return api.CancelMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) ExportMyData(ctx context.Context, _ api.ExportMyDataRequestObject) (api.ExportMyDataResponseObject, error) {
	return api.ExportMyData503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) GetMyDeletion(ctx context.Context, _ api.GetMyDeletionRequestObject) (api.GetMyDeletionResponseObject, error) {
	return api.GetMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) ScheduleMyDeletion(ctx context.Context, _ api.ScheduleMyDeletionRequestObject) (api.ScheduleMyDeletionResponseObject, error) {
	return api.ScheduleMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}
