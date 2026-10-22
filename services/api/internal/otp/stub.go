package otp

import (
	"context"
	"crypto/subtle"
	"fmt"
)

// DevCode is the only code the stub accepts.
const DevCode = "123456"

// Stub is the dev SMS provider. It sends nothing, logs nothing and accepts
// DevCode for any valid number.
type Stub struct{}

// NewStub returns the stub. It refuses to exist outside dev so a mis-set
// environment can never accept a fixed code in staging or prod.
func NewStub(env string) (Stub, error) {
	if env != "dev" {
		return Stub{}, fmt.Errorf("otp stub is dev only, APP_ENV is %q", env)
	}
	return Stub{}, nil
}

// Send succeeds for any valid number.
func (Stub) Send(_ context.Context, phone string) error {
	if !ValidPhone(phone) {
		return ErrInvalidNumber
	}
	return nil
}

// Check accepts DevCode only.
func (Stub) Check(_ context.Context, phone, code string) (bool, error) {
	if !ValidPhone(phone) {
		return false, ErrInvalidNumber
	}
	return subtle.ConstantTimeCompare([]byte(code), []byte(DevCode)) == 1, nil
}
