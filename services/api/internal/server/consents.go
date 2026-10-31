package server

import (
	"context"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithConsents turns on consent choices. Without it the routes answer 503.
func WithConsents(c repo.Consents) Option { return func(h *handlers) { h.consents = c } }

const msgNoConsents = "Privacy choices are not available right now."

func toAPIConsents(c model.Consents) api.Consents {
	out := api.Consents{
		Location: api.ConsentState(c.Location), Notifications: api.ConsentState(c.Notifications),
		Photos: api.ConsentState(c.Photos), Analytics: api.ConsentState(c.Analytics),
	}
	if !c.UpdatedAt.IsZero() {
		out.UpdatedAt = &c.UpdatedAt
	}
	return out
}

// GetMyConsents returns the caller's consent choices.
func (h *handlers) GetMyConsents(ctx context.Context, _ api.GetMyConsentsRequestObject) (api.GetMyConsentsResponseObject, error) {
	id, err := identity(ctx, "GetMyConsents")
	if err != nil {
		return nil, err
	}
	if h.consents == nil {
		return api.GetMyConsents503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoConsents))}, nil
	}
	c, err := h.consents.Get(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	return api.GetMyConsents200JSONResponse(toAPIConsents(c)), nil
}

// UpdateMyConsents changes the purposes that are sent.
func (h *handlers) UpdateMyConsents(ctx context.Context, req api.UpdateMyConsentsRequestObject) (api.UpdateMyConsentsResponseObject, error) {
	id, err := identity(ctx, "UpdateMyConsents")
	if err != nil {
		return nil, err
	}
	if h.consents == nil {
		return api.UpdateMyConsents503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoConsents))}, nil
	}
	b := req.Body
	pick := func(s *string) *model.ConsentState {
		if s == nil {
			return nil
		}
		c := model.ConsentState(*s)
		return &c
	}
	p := model.ConsentsPatch{Location: pick((*string)(b.Location)), Notifications: pick((*string)(b.Notifications)), Photos: pick((*string)(b.Photos)), Analytics: pick((*string)(b.Analytics))}
	c, err := h.consents.Update(ctx, id.UserID, p, time.Now())
	if err != nil {
		var invalid *model.ValidationError
		if errors.As(err, &invalid) {
			return api.UpdateMyConsents400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, invalid.Error())}, nil
		}
		return nil, err
	}
	return api.UpdateMyConsents200JSONResponse(toAPIConsents(c)), nil
}
