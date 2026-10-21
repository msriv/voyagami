package otp

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"
)

// Limits caps attempts per number inside a sliding window.
type Limits struct {
	SendMax  int
	CheckMax int
	Window   time.Duration
}

// DefaultLimits allows 5 codes and 10 checks per number per 15 minutes.
var DefaultLimits = Limits{SendMax: 5, CheckMax: 10, Window: 15 * time.Minute}

// Limited wraps a provider and rate limits Send and Check per number. State
// is in memory and per instance; a shared limiter can replace it later. Keys
// are hashes, so no phone number is held in a form that could be logged.
type Limited struct {
	next   SMSProvider
	limits Limits
	now    func() time.Time

	mu     sync.Mutex
	sends  map[[32]byte][]time.Time
	checks map[[32]byte][]time.Time
}

// NewLimited wraps next with limits.
func NewLimited(next SMSProvider, limits Limits) *Limited {
	return &Limited{
		next:   next,
		limits: limits,
		now:    time.Now,
		sends:  map[[32]byte][]time.Time{},
		checks: map[[32]byte][]time.Time{},
	}
}

// Send forwards to the wrapped provider unless the number is over its limit.
func (l *Limited) Send(ctx context.Context, phone string) error {
	if !ValidPhone(phone) {
		return ErrInvalidNumber
	}
	if !l.allow(l.sends, phone, l.limits.SendMax) {
		return ErrRateLimited
	}
	return l.next.Send(ctx, phone)
}

// Check forwards to the wrapped provider unless the number is over its limit.
func (l *Limited) Check(ctx context.Context, phone, code string) (bool, error) {
	if !ValidPhone(phone) {
		return false, ErrInvalidNumber
	}
	if !l.allow(l.checks, phone, l.limits.CheckMax) {
		return false, ErrRateLimited
	}
	return l.next.Check(ctx, phone, code)
}

// allow records an attempt and reports whether it is within limit.
func (l *Limited) allow(m map[[32]byte][]time.Time, phone string, limit int) bool {
	key := sha256.Sum256([]byte(phone))
	now := l.now()
	cutoff := now.Add(-l.limits.Window)

	l.mu.Lock()
	defer l.mu.Unlock()

	kept := m[key][:0]
	for _, t := range m[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		m[key] = kept
		return false
	}
	m[key] = append(kept, now)
	return true
}
