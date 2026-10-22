// Package auth defines who is calling the API. It holds the contract only: the
// bearer-token middleware lives in internal/server and the real token verifier
// arrives with phone sign-in.
package auth

import (
	"context"
	"errors"
)

// ErrInvalidToken means the token is missing, malformed, expired or revoked.
// Verifiers return it, or an error wrapping it, for any token they reject.
var ErrInvalidToken = errors.New("invalid token")

// Identity is the signed-in person. It carries no phone number.
type Identity struct {
	// UserID is the stable account ID.
	UserID string
	// DisplayName is the name the person chose, if any.
	DisplayName string
	// SessionID is the sign-in session the token belongs to, if the verifier
	// knows it. Sign-out ends this session.
	SessionID string
}

// Verifier turns a bearer token into an Identity. Any error other than
// ErrInvalidToken means verification could not run, not that the token is bad.
type Verifier interface {
	Verify(ctx context.Context, token string) (Identity, error)
}

// RejectAll is the verifier used until real sign-in exists: every token is
// invalid, so every protected route answers 401.
type RejectAll struct{}

// Verify always returns ErrInvalidToken.
func (RejectAll) Verify(context.Context, string) (Identity, error) {
	return Identity{}, ErrInvalidToken
}

type ctxKey struct{}

// NewContext returns ctx carrying id.
func NewContext(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the Identity set by the auth middleware.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}
