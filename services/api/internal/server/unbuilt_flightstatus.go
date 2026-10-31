package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// Not built yet: answers 503 until its ticket replaces it. Delete this file when the real code lands.

func (h *handlers) GetFlightStatus(ctx context.Context, _ api.GetFlightStatusRequestObject) (api.GetFlightStatusResponseObject, error) {
	return api.GetFlightStatus503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}
