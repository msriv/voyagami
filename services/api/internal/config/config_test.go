package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr []string
	}{
		{
			name: "defaults",
			want: Config{Env: "dev", Port: 8080, LogLevel: slog.LevelInfo, ShutdownTimeout: 10 * time.Second, RequestTimeout: 30 * time.Second},
		},
		{
			name: "all set",
			env:  map[string]string{"APP_ENV": "staging", "PORT": "9000", "LOG_LEVEL": "debug", "SHUTDOWN_TIMEOUT": "5s", "REQUEST_TIMEOUT": "1m", "AUTH_SECRET": strings.Repeat("s", 32), "GCP_PROJECT": "p"},
			want: Config{Env: "staging", Port: 9000, LogLevel: slog.LevelDebug, ShutdownTimeout: 5 * time.Second, RequestTimeout: time.Minute, AuthSecret: strings.Repeat("s", 32), GCPProject: "p"},
		},
		{
			name: "dev plan override",
			env:  map[string]string{"DEV_PLAN": "plus"},
			want: Config{Env: "dev", Port: 8080, LogLevel: slog.LevelInfo, ShutdownTimeout: 10 * time.Second, RequestTimeout: 30 * time.Second, DevPlan: "plus"},
		},
		{
			name:    "dev plan rejected outside dev",
			env:     map[string]string{"APP_ENV": "staging", "AUTH_SECRET": strings.Repeat("s", 32), "DEV_PLAN": "plus"},
			wantErr: []string{"DEV_PLAN"},
		},
		{
			name:    "dev plan must be known",
			env:     map[string]string{"DEV_PLAN": "gold"},
			wantErr: []string{"DEV_PLAN"},
		},
		{
			name:    "secret required outside dev",
			env:     map[string]string{"APP_ENV": "prod"},
			wantErr: []string{"AUTH_SECRET"},
		},
		{
			name:    "secret too short",
			env:     map[string]string{"AUTH_SECRET": "short"},
			wantErr: []string{"AUTH_SECRET"},
		},
		{
			name:    "bad env",
			env:     map[string]string{"APP_ENV": "qa"},
			wantErr: []string{"APP_ENV"},
		},
		{
			name:    "port not a number",
			env:     map[string]string{"PORT": "abc"},
			wantErr: []string{"PORT"},
		},
		{
			name:    "port out of range",
			env:     map[string]string{"PORT": "70000"},
			wantErr: []string{"PORT"},
		},
		{
			name:    "bad log level",
			env:     map[string]string{"LOG_LEVEL": "loud"},
			wantErr: []string{"LOG_LEVEL"},
		},
		{
			name:    "non-positive duration",
			env:     map[string]string{"SHUTDOWN_TIMEOUT": "0s"},
			wantErr: []string{"SHUTDOWN_TIMEOUT"},
		},
		{
			name:    "all problems reported together",
			env:     map[string]string{"APP_ENV": "qa", "PORT": "0", "REQUEST_TIMEOUT": "soon"},
			wantErr: []string{"APP_ENV", "PORT", "REQUEST_TIMEOUT"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(env(tt.env))
			if len(tt.wantErr) > 0 {
				if err == nil {
					t.Fatalf("want error mentioning %v, got none", tt.wantErr)
				}
				for _, s := range tt.wantErr {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("error %q does not mention %s", err, s)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
