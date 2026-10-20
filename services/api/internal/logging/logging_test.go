package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewUsesCloudLoggingFields(t *testing.T) {
	tests := []struct {
		name         string
		log          func(*slog.Logger)
		wantSeverity string
	}{
		{"debug", func(l *slog.Logger) { l.Debug("hello") }, "DEBUG"},
		{"info", func(l *slog.Logger) { l.Info("hello") }, "INFO"},
		{"warn", func(l *slog.Logger) { l.Warn("hello") }, "WARNING"},
		{"error", func(l *slog.Logger) { l.Error("hello") }, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.log(New(&buf, slog.LevelDebug, "dev"))

			var got map[string]any
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("log line is not JSON: %v", err)
			}
			if got["severity"] != tt.wantSeverity {
				t.Errorf("severity = %v, want %s", got["severity"], tt.wantSeverity)
			}
			if got["message"] != "hello" {
				t.Errorf("message = %v, want hello", got["message"])
			}
			if got["env"] != "dev" {
				t.Errorf("env = %v, want dev", got["env"])
			}
			for _, k := range []string{"timestamp"} {
				if _, ok := got[k]; !ok {
					t.Errorf("missing field %s", k)
				}
			}
			for _, k := range []string{"level", "msg", "time"} {
				if _, ok := got[k]; ok {
					t.Errorf("unexpected default field %s", k)
				}
			}
		})
	}
}

func TestNewHonoursLevel(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelWarn, "dev").Info("quiet")
	if buf.Len() != 0 {
		t.Errorf("info line written at warn level: %s", buf.String())
	}
}
