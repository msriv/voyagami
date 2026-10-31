package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// Not built yet: answers 503 until its ticket replaces it. Delete this file when the real code lands.

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
