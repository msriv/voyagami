package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/devlog"
)

type fakeDevLogs struct{ entries []devlog.Entry }

func (f *fakeDevLogs) Add(_ context.Context, e devlog.Entry) error {
	f.entries = append(f.entries, e)
	return nil
}

func (f *fakeDevLogs) List(_ context.Context, deviceID string, _ time.Time, limit int) ([]devlog.Entry, error) {
	var out []devlog.Entry
	for i := len(f.entries) - 1; i >= 0 && len(out) < limit; i-- {
		if deviceID == "" || f.entries[i].DeviceID == deviceID {
			out = append(out, f.entries[i])
		}
	}
	return out, nil
}

func TestDevLogs(t *testing.T) {
	store := &fakeDevLogs{}
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{}, WithDevLogs(store))

	// No sign-in needed; fields are trimmed.
	body := `{"deviceId":"d1","appVersion":"1.0","level":"fatal","message":"` + strings.Repeat("x", 5000) + `","route":"/trips","breadcrumbs":["a","b"]}`
	if rec := call(h, "POST", "/v1/dev/logs", "", body); rec.Code != 204 {
		t.Fatalf("post: %d %s", rec.Code, rec.Body)
	}
	call(h, "POST", "/v1/dev/logs", "", `{"deviceId":"d2","level":"error","message":"other"}`)
	if got := len([]rune(store.entries[0].Message)); got != devlog.MaxMessage {
		t.Errorf("message length %d", got)
	}

	rec := call(h, "GET", "/v1/dev/logs?deviceId=d1", "", "")
	var out struct{ Entries []devlog.Entry }
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != 200 || len(out.Entries) != 1 || out.Entries[0].Route != "/trips" {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	if rec := call(h, "GET", "/v1/dev/logs", "", ""); !strings.Contains(rec.Body.String(), `"other"`) {
		t.Errorf("all devices: %s", rec.Body)
	}

	for name, b := range map[string]string{
		"no device":  `{"level":"error","message":"m"}`,
		"bad level":  `{"deviceId":"d","level":"loud","message":"m"}`,
		"no message": `{"deviceId":"d","level":"error"}`,
		"not json":   `nope`,
	} {
		if rec := call(h, "POST", "/v1/dev/logs", "", b); rec.Code != 400 {
			t.Errorf("%s: %d", name, rec.Code)
		}
	}
	if rec := call(h, "GET", "/v1/dev/logs?since=yesterday", "", ""); rec.Code != 400 {
		t.Errorf("bad since: %d", rec.Code)
	}
}

func TestDevLogsRateLimit(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{}, WithDevLogs(&fakeDevLogs{}))
	limited := false
	for range 40 {
		if call(h, "POST", "/v1/dev/logs", "", `{"deviceId":"d","level":"info","message":"m"}`).Code == 429 {
			limited = true
		}
	}
	if !limited {
		t.Error("never rate limited")
	}
}

func TestDevLogsOffByDefault(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{})
	if rec := call(h, "POST", "/v1/dev/logs", "", `{}`); rec.Code == 204 || rec.Code == 400 {
		t.Errorf("sink answered without being wired: %d", rec.Code)
	}
}
