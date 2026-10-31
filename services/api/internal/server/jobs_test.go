package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeJob struct {
	n   int
	err error
}

func (f fakeJob) Run(context.Context) (int, error) { return f.n, f.err }

func TestTrialReminderRoute(t *testing.T) {
	mux := http.NewServeMux()
	h := &handlers{}
	WithJobs(fakeJob{n: 3}, "s3cret")(h)
	h.mountJobs(mux)
	call := func(auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/jobs/trial-reminders", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	if rec := call(""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no secret: %d", rec.Code)
	}
	if rec := call("Bearer wrong"); rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong secret: %d", rec.Code)
	}
	rec := call("Bearer s3cret")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"reached":3`) {
		t.Errorf("good call: %d %s", rec.Code, rec.Body)
	}
	h.trialJob = fakeJob{err: errors.New("boom")}
	if rec := call("Bearer s3cret"); rec.Code != http.StatusInternalServerError {
		t.Errorf("failed run: %d", rec.Code)
	}
}
