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

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/auth"
)

// fakeVerifier accepts one token and can simulate a verifier outage.
type fakeVerifier struct {
	token string
	id    auth.Identity
	err   error
}

func (f fakeVerifier) Verify(_ context.Context, token string) (auth.Identity, error) {
	if f.err != nil {
		return auth.Identity{}, f.err
	}
	if token != f.token {
		return auth.Identity{}, auth.ErrInvalidToken
	}
	return f.id, nil
}

func quiet() *slog.Logger { return slog.New(slog.DiscardHandler) }

func do(h http.Handler, method, target string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) api.Error {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type = %q, want application/json (body %q)", ct, rec.Body.String())
	}
	var e api.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if e.Message == "" {
		t.Errorf("error has no message: %+v", e)
	}
	return e
}

func TestMeAuth(t *testing.T) {
	good := fakeVerifier{token: "good", id: auth.Identity{UserID: "u_1", DisplayName: "Asha"}}
	tests := []struct {
		name       string
		verifier   auth.Verifier
		header     map[string]string
		wantStatus int
		wantCode   string
	}{
		{"valid token", good, map[string]string{"Authorization": "Bearer good"}, 200, ""},
		{"scheme is case-insensitive", good, map[string]string{"Authorization": "bearer good"}, 200, ""},
		{"no header", good, nil, 401, codeUnauthenticated},
		{"wrong scheme", good, map[string]string{"Authorization": "Basic good"}, 401, codeUnauthenticated},
		{"empty token", good, map[string]string{"Authorization": "Bearer "}, 401, codeUnauthenticated},
		{"bad token", good, map[string]string{"Authorization": "Bearer nope"}, 401, codeUnauthenticated},
		{"reject-all verifier", auth.RejectAll{}, map[string]string{"Authorization": "Bearer good"}, 401, codeUnauthenticated},
		{"verifier outage", fakeVerifier{err: errors.New("down")}, map[string]string{"Authorization": "Bearer good"}, 503, codeUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(NewHandler(quiet(), tt.verifier), http.MethodGet, "/v1/me", tt.header)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode == "" {
				var me api.Me
				if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
					t.Fatal(err)
				}
				if me.Id != "u_1" || me.DisplayName == nil || *me.DisplayName != "Asha" {
					t.Errorf("me = %+v, want u_1 / Asha", me)
				}
				return
			}
			if got := decodeError(t, rec).Code; got != tt.wantCode {
				t.Errorf("code = %q, want %q", got, tt.wantCode)
			}
			if tt.wantStatus == 401 && rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestHealthzNeedsNoToken(t *testing.T) {
	rec := do(NewHandler(quiet(), auth.RejectAll{}), http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestRequestID(t *testing.T) {
	h := NewHandler(quiet(), auth.RejectAll{})

	t.Run("generated when absent", func(t *testing.T) {
		rec := do(h, http.MethodGet, "/healthz", nil)
		if got := rec.Header().Get(requestIDHeader); len(got) != 32 {
			t.Errorf("generated id = %q, want 32 hex chars", got)
		}
	})
	t.Run("kept when valid, and in error body", func(t *testing.T) {
		rec := do(h, http.MethodGet, "/v1/me", map[string]string{requestIDHeader: "abc-123.x_Y"})
		if got := rec.Header().Get(requestIDHeader); got != "abc-123.x_Y" {
			t.Errorf("id = %q, want the caller's id", got)
		}
		e := decodeError(t, rec)
		if e.RequestId == nil || *e.RequestId != "abc-123.x_Y" {
			t.Errorf("body requestId = %v, want abc-123.x_Y", e.RequestId)
		}
	})
	t.Run("replaced when unsafe", func(t *testing.T) {
		for _, bad := range []string{"has space", "new\nline", strings.Repeat("a", 65), "semi;colon"} {
			rec := do(h, http.MethodGet, "/healthz", map[string]string{requestIDHeader: bad})
			if got := rec.Header().Get(requestIDHeader); got == bad || len(got) != 32 {
				t.Errorf("unsafe id %q was kept as %q", bad, got)
			}
		}
	})
}

func TestRequestLogCarriesRequestID(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(slog.New(slog.NewJSONHandler(&buf, nil)), auth.RejectAll{})
	do(h, http.MethodGet, "/healthz", map[string]string{requestIDHeader: "req-1"})

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["requestId"] != "req-1" {
		t.Errorf("requestId = %v, want req-1", got["requestId"])
	}
}

func TestRouterErrorsUseErrorShape(t *testing.T) {
	h := NewHandler(quiet(), auth.RejectAll{})
	tests := []struct {
		method, target string
		wantStatus     int
		wantCode       string
	}{
		{http.MethodGet, "/nope", 404, codeNotFound},
		{http.MethodPost, "/healthz", 405, codeMethodNotAllowed},
	}
	for _, tt := range tests {
		rec := do(h, tt.method, tt.target, nil)
		if rec.Code != tt.wantStatus {
			t.Fatalf("%s %s status = %d, want %d", tt.method, tt.target, rec.Code, tt.wantStatus)
		}
		if got := decodeError(t, rec).Code; got != tt.wantCode {
			t.Errorf("%s %s code = %q, want %q", tt.method, tt.target, got, tt.wantCode)
		}
	}
	if rec := do(h, http.MethodPost, "/healthz", nil); rec.Header().Get("Allow") == "" {
		t.Error("405 lost its Allow header")
	}
}

func TestPanicIsRecovered(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret phone +911234567890")
	})
	h := withRequestID(logRequests(logger, recoverPanics(logger, boom)))

	rec := do(h, http.MethodGet, "/boom", map[string]string{requestIDHeader: "req-9"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeError(t, rec).Code; got != codeInternal {
		t.Errorf("code = %q, want %q", got, codeInternal)
	}
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(buf.String(), "911234567890") {
		t.Errorf("panic value leaked.\nbody: %s\nlog: %s", rec.Body.String(), buf.String())
	}
	if !strings.Contains(buf.String(), `"panicType":"string"`) || !strings.Contains(buf.String(), `"status":500`) {
		t.Errorf("log missing panic type or 500 status: %s", buf.String())
	}
}

// publicRoutes must match the operations the spec marks `security: []`.
func TestPublicRoutesMatchSpec(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile("../../../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			secured := doc.Security
			if op.Security != nil {
				secured = *op.Security
			}
			if len(secured) == 0 {
				want[method+" "+path] = true
			}
		}
	}
	if len(want) != len(publicRoutes) {
		t.Errorf("public routes = %v, spec says %v", publicRoutes, want)
	}
	for k := range want {
		if !publicRoutes[k] {
			t.Errorf("%s is public in the spec but protected in the server", k)
		}
	}
}
