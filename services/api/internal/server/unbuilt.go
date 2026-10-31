package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// These routes are in the API contract before their code lands, so the app can
// generate its client. Each answers 503 until its ticket replaces it; delete the
// method here when you build the real one.

const msgNotBuilt = "This is not available yet."

func (h *handlers) CancelMyDeletion(ctx context.Context, _ api.CancelMyDeletionRequestObject) (api.CancelMyDeletionResponseObject, error) {
	return api.CancelMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) CreateAskDraft(ctx context.Context, _ api.CreateAskDraftRequestObject) (api.CreateAskDraftResponseObject, error) {
	return api.CreateAskDraft503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) ExportMyData(ctx context.Context, _ api.ExportMyDataRequestObject) (api.ExportMyDataResponseObject, error) {
	return api.ExportMyData503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) GetFlightStatus(ctx context.Context, _ api.GetFlightStatusRequestObject) (api.GetFlightStatusResponseObject, error) {
	return api.GetFlightStatus503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) GetMyConsents(ctx context.Context, _ api.GetMyConsentsRequestObject) (api.GetMyConsentsResponseObject, error) {
	return api.GetMyConsents503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) GetMyDeletion(ctx context.Context, _ api.GetMyDeletionRequestObject) (api.GetMyDeletionResponseObject, error) {
	return api.GetMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) GetSponsoredCard(ctx context.Context, _ api.GetSponsoredCardRequestObject) (api.GetSponsoredCardResponseObject, error) {
	return api.GetSponsoredCard503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) HideSponsoredCard(ctx context.Context, _ api.HideSponsoredCardRequestObject) (api.HideSponsoredCardResponseObject, error) {
	return api.HideSponsoredCard503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) RecordSponsoredClick(ctx context.Context, _ api.RecordSponsoredClickRequestObject) (api.RecordSponsoredClickResponseObject, error) {
	return api.RecordSponsoredClick503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) RecordSponsoredImpression(ctx context.Context, _ api.RecordSponsoredImpressionRequestObject) (api.RecordSponsoredImpressionResponseObject, error) {
	return api.RecordSponsoredImpression503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) ScheduleMyDeletion(ctx context.Context, _ api.ScheduleMyDeletionRequestObject) (api.ScheduleMyDeletionResponseObject, error) {
	return api.ScheduleMyDeletion503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) SendEvents(ctx context.Context, _ api.SendEventsRequestObject) (api.SendEventsResponseObject, error) {
	return api.SendEvents503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) UpdateMyConsents(ctx context.Context, _ api.UpdateMyConsentsRequestObject) (api.UpdateMyConsentsResponseObject, error) {
	return api.UpdateMyConsents503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}
