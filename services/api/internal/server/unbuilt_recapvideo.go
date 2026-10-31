package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// Not built yet: answers 503 until its ticket replaces it. Delete this file when the real code lands.

func (h *handlers) CreateRecapVideo(ctx context.Context, _ api.CreateRecapVideoRequestObject) (api.CreateRecapVideoResponseObject, error) {
	return api.CreateRecapVideo503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) GetRecapVideo(ctx context.Context, _ api.GetRecapVideoRequestObject) (api.GetRecapVideoResponseObject, error) {
	return api.GetRecapVideo503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}
