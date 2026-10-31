package server

import (
	"context"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithDevices turns on push token registration. Without it the routes answer 503.
func WithDevices(d repo.Devices) Option { return func(h *handlers) { h.devices = d } }

const msgNoDevices = "Push notifications are not available right now."

// RegisterDevice stores or refreshes the caller's push token for one device.
func (h *handlers) RegisterDevice(ctx context.Context, req api.RegisterDeviceRequestObject) (api.RegisterDeviceResponseObject, error) {
	id, err := identity(ctx, "RegisterDevice")
	if err != nil {
		return nil, err
	}
	if h.devices == nil {
		return api.RegisterDevice503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoDevices))}, nil
	}
	d := model.Device{ID: req.DeviceId, Platform: model.Platform(req.Body.Platform), PushToken: req.Body.PushToken, LastSeenAt: time.Now()}
	if err := h.devices.Put(ctx, id.UserID, d); err != nil {
		var invalid *model.ValidationError
		if errors.As(err, &invalid) {
			return api.RegisterDevice400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, invalid.Error())}, nil
		}
		return nil, err
	}
	return api.RegisterDevice204Response{}, nil
}

// RemoveDevice stops pushes to one of the caller's devices.
func (h *handlers) RemoveDevice(ctx context.Context, req api.RemoveDeviceRequestObject) (api.RemoveDeviceResponseObject, error) {
	id, err := identity(ctx, "RemoveDevice")
	if err != nil {
		return nil, err
	}
	if h.devices == nil {
		return api.RemoveDevice503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoDevices))}, nil
	}
	if err := h.devices.Delete(ctx, id.UserID, req.DeviceId); err != nil {
		return nil, err
	}
	return api.RemoveDevice204Response{}, nil
}
