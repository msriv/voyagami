package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/auth"
	"github.com/msriv/voyagami/services/api/internal/otp"
	"github.com/msriv/voyagami/services/api/internal/signin"
)

type fakeSignIn struct {
	requestErr, verifyErr, refreshErr, signOutErr error
	tokens                                        signin.Tokens
	gotDevice                                     signin.Device
	signedOut                                     string
}

func (f *fakeSignIn) RequestCode(context.Context, string) error { return f.requestErr }

func (f *fakeSignIn) Verify(_ context.Context, _, _ string, d signin.Device) (signin.Tokens, error) {
	f.gotDevice = d
	return f.tokens, f.verifyErr
}

func (f *fakeSignIn) Refresh(context.Context, string) (signin.Tokens, error) {
	return f.tokens, f.refreshErr
}

func (f *fakeSignIn) SignOut(_ context.Context, sid string) error {
	f.signedOut = sid
	return f.signOutErr
}

func post(h http.Handler, target, body string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRequestOtp(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		body     string
		want     int
		wantCode string
	}{
		{"sent", nil, `{"phone":"+14155550123"}`, 204, ""},
		{"bad number", otp.ErrInvalidNumber, `{"phone":"x"}`, 400, codeBadRequest},
		{"rate limited", otp.ErrRateLimited, `{"phone":"+14155550123"}`, 429, codeRateLimited},
		{"unreadable body", nil, `{`, 400, codeBadRequest},
		{"provider down", errors.New("boom"), `{"phone":"+14155550123"}`, 500, codeInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(quiet(), auth.RejectAll{}, WithSignIn(&fakeSignIn{requestErr: tt.err}))
			rec := post(h, "/v1/auth/otp/request", tt.body, nil)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.want, rec.Body.String())
			}
			if tt.wantCode != "" {
				if got := decodeError(t, rec).Code; got != tt.wantCode {
					t.Errorf("code = %q, want %q", got, tt.wantCode)
				}
			}
		})
	}
}

func TestVerifyOtp(t *testing.T) {
	tokens := signin.Tokens{AccessToken: "a", RefreshToken: "r", ExpiresIn: 15 * time.Minute, UserID: "u1", IsNewUser: true}

	t.Run("signed in", func(t *testing.T) {
		f := &fakeSignIn{tokens: tokens}
		h := NewHandler(quiet(), auth.RejectAll{}, WithSignIn(f))
		rec := post(h, "/v1/auth/otp/verify", `{"phone":"+14155550123","code":"123456","region":"IN","language":"hi"}`, nil)
		if rec.Code != 200 {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		var got api.SessionTokens
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		want := api.SessionTokens{AccessToken: "a", RefreshToken: "r", ExpiresIn: 900, UserId: "u1", IsNewUser: true}
		if got != want {
			t.Errorf("body = %+v, want %+v", got, want)
		}
		if f.gotDevice != (signin.Device{Region: "IN", Language: "hi"}) {
			t.Errorf("device = %+v", f.gotDevice)
		}
	})

	errCases := []struct {
		name     string
		err      error
		body     string
		want     int
		wantCode string
	}{
		{"wrong code", signin.ErrInvalidCode, `{"phone":"+14155550123","code":"0"}`, 400, codeInvalidCode},
		{"bad number", otp.ErrInvalidNumber, `{"phone":"x","code":"0"}`, 400, codeBadRequest},
		{"rate limited", otp.ErrRateLimited, `{"phone":"+14155550123","code":"0"}`, 429, codeRateLimited},
		{"missing code", nil, `{"phone":"+14155550123"}`, 400, codeBadRequest},
	}
	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(quiet(), auth.RejectAll{}, WithSignIn(&fakeSignIn{verifyErr: tt.err, tokens: tokens}))
			rec := post(h, "/v1/auth/otp/verify", tt.body, nil)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.want, rec.Body.String())
			}
			if got := decodeError(t, rec).Code; got != tt.wantCode {
				t.Errorf("code = %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestRefreshAndSignOut(t *testing.T) {
	f := &fakeSignIn{tokens: signin.Tokens{AccessToken: "a2", RefreshToken: "r2", ExpiresIn: time.Minute, UserID: "u1"}}
	verifier := fakeVerifier{token: "good", id: auth.Identity{UserID: "u1", SessionID: "s1"}}
	h := NewHandler(quiet(), verifier, WithSignIn(f))

	if rec := post(h, "/v1/auth/refresh", `{"refreshToken":"r1"}`, nil); rec.Code != 200 {
		t.Fatalf("refresh status = %d: %s", rec.Code, rec.Body.String())
	}
	f.refreshErr = auth.ErrInvalidToken
	rec := post(h, "/v1/auth/refresh", `{"refreshToken":"r1"}`, nil)
	if rec.Code != 401 || decodeError(t, rec).Code != codeUnauthenticated {
		t.Fatalf("reused refresh = %d %s", rec.Code, rec.Body.String())
	}

	if rec := post(h, "/v1/auth/sign-out", "", nil); rec.Code != 401 {
		t.Errorf("sign-out without token = %d, want 401", rec.Code)
	}
	rec = post(h, "/v1/auth/sign-out", "", map[string]string{"Authorization": "Bearer good"})
	if rec.Code != 204 {
		t.Fatalf("sign-out = %d: %s", rec.Code, rec.Body.String())
	}
	if f.signedOut != "s1" {
		t.Errorf("signed out session %q, want s1", f.signedOut)
	}
}

func TestSignInUnavailableWithoutService(t *testing.T) {
	h := NewHandler(quiet(), auth.RejectAll{})
	for _, p := range []string{"/v1/auth/otp/request", "/v1/auth/otp/verify"} {
		rec := post(h, p, `{"phone":"+14155550123","code":"1"}`, nil)
		if rec.Code != 503 || decodeError(t, rec).Code != codeUnavailable {
			t.Errorf("%s = %d %s, want 503 unavailable", p, rec.Code, rec.Body.String())
		}
	}
}

func TestSignInLogsHoldNoPhoneOrCode(t *testing.T) {
	var buf bytes.Buffer
	f := &fakeSignIn{verifyErr: errors.New("provider failed"), requestErr: errors.New("provider failed")}
	h := NewHandler(slog.New(slog.NewJSONHandler(&buf, nil)), auth.RejectAll{}, WithSignIn(f))
	post(h, "/v1/auth/otp/request", `{"phone":"+14155550123"}`, nil)
	post(h, "/v1/auth/otp/verify", `{"phone":"+14155550123","code":"654321"}`, nil)
	if out := buf.String(); strings.Contains(out, "4155550123") || strings.Contains(out, "654321") {
		t.Errorf("log leaks phone or code: %s", out)
	}
}
