package server

import (
	"bytes"
	"context"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/export"
)

// WithExport turns on the data download. Without it the route answers 503.
func WithExport(s export.Sources) Option { return func(h *handlers) { h.export = &s } }

const msgNoExport = "Data download is not available right now."

// ExportMyData returns a ZIP of the caller's own data. Free on every plan.
func (h *handlers) ExportMyData(ctx context.Context, _ api.ExportMyDataRequestObject) (api.ExportMyDataResponseObject, error) {
	id, err := identity(ctx, "ExportMyData")
	if err != nil {
		return nil, err
	}
	if h.export == nil {
		return api.ExportMyData503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoExport))}, nil
	}
	zip, err := export.Build(ctx, *h.export, id.UserID)
	if err != nil {
		return nil, err
	}
	return api.ExportMyData200ApplicationzipResponse{Body: bytes.NewReader(zip), ContentLength: int64(len(zip))}, nil
}
