package otp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const phone = "+919876543210"

func TestValidPhone(t *testing.T) {
	for _, s := range []string{"+919876543210", "+14155550123", "+4420712345678"} {
		if !ValidPhone(s) {
			t.Errorf("ValidPhone(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "9876543210", "+0123456789", "+12", "+1234567890123456", "+91 98765 43210", "+91abc4567890"} {
		if ValidPhone(s) {
			t.Errorf("ValidPhone(%q) = true, want false", s)
		}
	}
}

func TestNewStubDevOnly(t *testing.T) {
	if _, err := NewStub("dev"); err != nil {
		t.Fatalf("dev: %v", err)
	}
	for _, env := range []string{"staging", "prod", ""} {
		if _, err := NewStub(env); err == nil {
			t.Errorf("NewStub(%q) succeeded, want error", env)
		}
	}
}

func TestStub(t *testing.T) {
	s, _ := NewStub("dev")
	ctx := context.Background()

	if err := s.Send(ctx, phone); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := s.Send(ctx, "nope"); !errors.Is(err, ErrInvalidNumber) {
		t.Errorf("Send bad number = %v, want ErrInvalidNumber", err)
	}
	if ok, err := s.Check(ctx, phone, DevCode); err != nil || !ok {
		t.Errorf("Check right code = %v, %v", ok, err)
	}
	if ok, err := s.Check(ctx, phone, "000000"); err != nil || ok {
		t.Errorf("Check wrong code = %v, %v", ok, err)
	}
	if _, err := s.Check(ctx, "nope", DevCode); !errors.Is(err, ErrInvalidNumber) {
		t.Errorf("Check bad number = %v, want ErrInvalidNumber", err)
	}
}

func TestLimited(t *testing.T) {
	s, _ := NewStub("dev")
	l := NewLimited(s, Limits{SendMax: 2, CheckMax: 2, Window: time.Minute})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := l.Send(ctx, phone); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	if err := l.Send(ctx, phone); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("third send = %v, want ErrRateLimited", err)
	}
	// Another number is unaffected.
	if err := l.Send(ctx, "+14155550123"); err != nil {
		t.Errorf("other number: %v", err)
	}
	// Checks have their own budget.
	for i := 0; i < 2; i++ {
		if _, err := l.Check(ctx, phone, "000000"); err != nil {
			t.Fatalf("check %d: %v", i, err)
		}
	}
	if _, err := l.Check(ctx, phone, DevCode); !errors.Is(err, ErrRateLimited) {
		t.Errorf("third check = %v, want ErrRateLimited", err)
	}
	// The window slides.
	now = now.Add(time.Minute + time.Second)
	if err := l.Send(ctx, phone); err != nil {
		t.Errorf("send after window: %v", err)
	}
	if ok, err := l.Check(ctx, phone, DevCode); err != nil || !ok {
		t.Errorf("check after window = %v, %v", ok, err)
	}
}

func TestErrorsLeakNothing(t *testing.T) {
	_, stubErr := NewStub("prod")
	for _, err := range []error{ErrRateLimited, ErrInvalidNumber, stubErr} {
		if strings.Contains(err.Error(), phone) || strings.Contains(err.Error(), DevCode) {
			t.Errorf("error leaks data: %v", err)
		}
	}
}
