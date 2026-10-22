package server

import (
	"context"
	"errors"
	"strings"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/otp"
	"github.com/msriv/voyagami/services/api/internal/signin"
)

// SignIn is the sign-in behaviour the routes need. signin.Service implements it.
type SignIn interface {
	RequestCode(ctx context.Context, phone string) error
	Verify(ctx context.Context, phone, code string, dev signin.Device) (signin.Tokens, error)
	Refresh(ctx context.Context, refreshToken string) (signin.Tokens, error)
	SignOut(ctx context.Context, sessionID string) error
}

// Option adds a service to the handler.
type Option func(*handlers)

// WithSignIn turns on the sign-in routes. Without it they answer 503.
func WithSignIn(s SignIn) Option { return func(h *handlers) { h.signIn = s } }

func errBody(ctx context.Context, code, message string) api.Error {
	e := api.Error{Code: code, Message: message}
	if id := requestIDFrom(ctx); id != "" {
		e.RequestId = &id
	}
	return e
}

const msgNoSignIn = "Sign-in is not available right now."

// RequestOtp sends a one-time code.
func (h *handlers) RequestOtp(ctx context.Context, req api.RequestOtpRequestObject) (api.RequestOtpResponseObject, error) {
	if h.signIn == nil {
		return api.RequestOtp503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSignIn))}, nil
	}
	err := h.signIn.RequestCode(ctx, req.Body.Phone)
	switch {
	case err == nil:
		return api.RequestOtp204Response{}, nil
	case errors.Is(err, otp.ErrInvalidNumber):
		return api.RequestOtp400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, "Enter the phone number with its country code, like +14155550123."))}, nil
	case errors.Is(err, otp.ErrRateLimited):
		return api.RequestOtp429JSONResponse{TooManyRequestsJSONResponse: api.TooManyRequestsJSONResponse(errBody(ctx, codeRateLimited, "Too many codes requested. Try again later."))}, nil
	}
	return nil, err
}

// VerifyOtp checks a code and signs the person in.
func (h *handlers) VerifyOtp(ctx context.Context, req api.VerifyOtpRequestObject) (api.VerifyOtpResponseObject, error) {
	if h.signIn == nil {
		return api.VerifyOtp503JSONResponse{UnavailableJSONResponse: api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoSignIn))}, nil
	}
	b := req.Body
	var dev signin.Device
	if b.Region != nil {
		dev.Region = *b.Region
	}
	if b.Language != nil {
		dev.Language = strings.TrimSpace(*b.Language)
	}
	if b.Code == "" || len(dev.Region) > 2 || len(dev.Region) == 1 || len(dev.Language) > 35 {
		return api.VerifyOtp400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, "The request has a missing or invalid field."))}, nil
	}
	t, err := h.signIn.Verify(ctx, b.Phone, b.Code, dev)
	switch {
	case err == nil:
		return api.VerifyOtp200JSONResponse(sessionTokens(t)), nil
	case errors.Is(err, otp.ErrInvalidNumber):
		return api.VerifyOtp400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeBadRequest, "Enter the phone number with its country code, like +14155550123."))}, nil
	case errors.Is(err, signin.ErrInvalidCode):
		return api.VerifyOtp400JSONResponse{BadRequestJSONResponse: api.BadRequestJSONResponse(errBody(ctx, codeInvalidCode, "That code is not right."))}, nil
	case errors.Is(err, otp.ErrRateLimited):
		return api.VerifyOtp429JSONResponse{TooManyRequestsJSONResponse: api.TooManyRequestsJSONResponse(errBody(ctx, codeRateLimited, "Too many attempts. Try again later."))}, nil
	}
	return nil, err
}

// RefreshSession swaps a refresh token for a new pair.
func (h *handlers) RefreshSession(ctx context.Context, req api.RefreshSessionRequestObject) (api.RefreshSessionResponseObject, error) {
	if h.signIn == nil {
		return api.RefreshSession401JSONResponse{UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(errBody(ctx, codeUnauthenticated, msgNoSignIn))}, nil
	}
	t, err := h.signIn.Refresh(ctx, req.Body.RefreshToken)
	switch {
	case err == nil:
		return api.RefreshSession200JSONResponse(sessionTokens(t)), nil
	case errors.Is(err, auth.ErrInvalidToken):
		return api.RefreshSession401JSONResponse{UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(errBody(ctx, codeUnauthenticated, "The refresh token is invalid or already used. Sign in again."))}, nil
	}
	return nil, err
}

// SignOut ends the caller's session.
func (h *handlers) SignOut(ctx context.Context, _ api.SignOutRequestObject) (api.SignOutResponseObject, error) {
	id, ok := auth.FromContext(ctx)
	if !ok {
		return nil, errors.New("SignOut reached without an identity")
	}
	if h.signIn == nil {
		return api.SignOut204Response{}, nil
	}
	if err := h.signIn.SignOut(ctx, id.SessionID); err != nil {
		return nil, err
	}
	return api.SignOut204Response{}, nil
}

func sessionTokens(t signin.Tokens) api.SessionTokens {
	return api.SessionTokens{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresIn:    int(t.ExpiresIn.Seconds()),
		UserId:       t.UserID,
		IsNewUser:    t.IsNewUser,
	}
}
