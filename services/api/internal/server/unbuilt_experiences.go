package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// Not built yet: answers 503 until its ticket replaces it. Delete this file when the real code lands.

func (h *handlers) ListExperiences(ctx context.Context, _ api.ListExperiencesRequestObject) (api.ListExperiencesResponseObject, error) {
	return api.ListExperiences503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}

func (h *handlers) RecordExperienceClick(ctx context.Context, _ api.RecordExperienceClickRequestObject) (api.RecordExperienceClickResponseObject, error) {
	return api.RecordExperienceClick503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}
