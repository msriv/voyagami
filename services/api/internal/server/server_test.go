package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/auth"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantBody   string
	}{
		{"get", http.MethodGet, "/health", http.StatusOK, `{"status":"ok"}`},
		{"get with query", http.MethodGet, "/health?x=1", http.StatusOK, `{"status":"ok"}`},
		{"post not allowed", http.MethodPost, "/health", http.StatusMethodNotAllowed, ""},
		{"unknown route", http.MethodGet, "/nope", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(slog.New(slog.DiscardHandler), auth.RejectAll{})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.target, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" {
				if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
					t.Errorf("body = %s, want %s", got, tt.wantBody)
				}
				if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("content type = %q, want application/json", ct)
				}
			}
		})
	}
}

func TestRequestLogHasNoQueryString(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(slog.New(slog.NewJSONHandler(&buf, nil)), auth.RejectAll{})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health?phone=%2B911234567890", nil))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("log line is not JSON: %v", err)
	}
	if got["path"] != "/health" {
		t.Errorf("path = %v, want /health", got["path"])
	}
	if got["status"] != float64(200) {
		t.Errorf("status = %v, want 200", got["status"])
	}
	if strings.Contains(buf.String(), "911234567890") {
		t.Errorf("log line leaked the query string: %s", buf.String())
	}
}
