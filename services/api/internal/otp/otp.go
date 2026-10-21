// Package otp holds the phone-code contract. Real SMS providers (MSG91 for
// India, another provider elsewhere) arrive in P1 and implement SMSProvider;
// until then dev uses Stub.
package otp

import (
	"context"
	"errors"
)

var (
	// ErrRateLimited means too many codes were requested or checked for one
	// number. Callers answer 429.
	ErrRateLimited = errors.New("too many attempts")
	// ErrInvalidNumber means the number is not a usable E.164 phone number.
	ErrInvalidNumber = errors.New("invalid phone number")
)

// SMSProvider sends one-time codes and checks them. Providers own code
// generation and expiry, as hosted verify APIs do.
//
// Implementations must not log or return phone numbers or codes in errors.
type SMSProvider interface {
	// Send starts a verification for phone (E.164, for example +919876543210).
	Send(ctx context.Context, phone string) error
	// Check reports whether code is right for phone. A wrong code returns
	// (false, nil). An error means the check could not run.
	Check(ctx context.Context, phone, code string) (bool, error)
}
