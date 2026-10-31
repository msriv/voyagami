package server

import (
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// Not built yet: answers 503 until its ticket replaces it. Delete this file when the real code lands.

func (h *handlers) SendEvents(ctx context.Context, _ api.SendEventsRequestObject) (api.SendEventsResponseObject, error) {
	return api.SendEvents503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNotBuilt))}, nil
}
