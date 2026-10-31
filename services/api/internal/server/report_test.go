package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/analytics"
)

type fakeRows struct{ since string }

func (f *fakeRows) Rows(_ context.Context, since string, _ int) ([]analytics.Row, error) {
	f.since = since
	return []analytics.Row{{UID: "a", Name: "app_open"}, {UID: "a", Name: "sign_in"}}, nil
}

func TestAnalyticsReport(t *testing.T) {
	rows := &fakeRows{}
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{}, WithAnalyticsReport(rows, "s3cret"))
	get := func(path, auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if c := get("/v1/jobs/analytics-report", "").Code; c != 401 {
		t.Errorf("no secret: %d", c)
	}
	if c := get("/v1/jobs/analytics-report", "Bearer wrong").Code; c != 401 {
		t.Errorf("wrong secret: %d", c)
	}
	rec := get("/v1/jobs/analytics-report?days=7", "Bearer s3cret")
	var r analytics.Report
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	if rec.Code != 200 || len(r.Events) != 2 || r.Since != rows.since || r.Since == "" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if c := get("/v1/jobs/analytics-report?days=0", "Bearer s3cret").Code; c != 400 {
		t.Errorf("bad days: %d", c)
	}
}
