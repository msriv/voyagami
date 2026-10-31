// Package config reads and validates the API's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// Config holds every setting the API needs. Secrets are never defaults here.
type Config struct {
	Env             string
	Port            int
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	RequestTimeout  time.Duration
	// AuthSecret signs sign-in tokens and keys the phone hash. Required
	// outside dev; in dev an empty value means a random per-process secret.
	AuthSecret string
	// GCPProject is the Firestore project. Empty means detect it from the
	// environment (Cloud Run) or use the emulator's project.
	GCPProject string
	// DevPlan forces every account onto "free" or "plus" so plan gates can be
	// tried in dev. Empty means no override. Rejected outside dev.
	DevPlan string
	// VertexLocation is the Vertex AI region used to read bookings. Empty
	// turns booking import off. An empty VertexModel means the importer's default.
	VertexLocation string
	VertexModel    string
	// PlacesAPIKey is the Google Places key (VOY-7). Empty means dev falls back to OpenStreetMap search.
	PlacesAPIKey string
	// RevenueCatWebhookSecret is the Authorization value RevenueCat sends with
	// each webhook. Empty turns the webhook off.
	RevenueCatWebhookSecret string
	// JobsSecret is the bearer secret the scheduler sends to /v1/jobs/*. Empty turns the jobs off.
	JobsSecret string
	// StayPartners is the JSON list of stay partners (see docs/stays.md). Empty means none.
	StayPartners string
	// ExperiencePartners is the JSON list of experience partners, same shape as StayPartners. Empty means none.
	ExperiencePartners string
}

// minAuthSecret is the shortest AUTH_SECRET accepted.
const minAuthSecret = 32

const (
	defaultVertexModel     = "gemini-2.5-flash"
	defaultPort            = 8080
	defaultShutdownTimeout = 10 * time.Second
	defaultRequestTimeout  = 30 * time.Second
)

var environments = map[string]bool{"dev": true, "staging": true, "prod": true}

// Load builds a Config from getenv (os.Getenv in production). Every problem
// is reported together so a bad deploy shows all of them at once.
func Load(getenv func(string) string) (Config, error) {
	var errs []error
	cfg := Config{
		Env:             "dev",
		Port:            defaultPort,
		LogLevel:        slog.LevelInfo,
		ShutdownTimeout: defaultShutdownTimeout,
		RequestTimeout:  defaultRequestTimeout,
	}

	if v := getenv("APP_ENV"); v != "" {
		if !environments[v] {
			errs = append(errs, fmt.Errorf("APP_ENV %q must be dev, staging or prod", v))
		} else {
			cfg.Env = v
		}
	}

	// Cloud Run sets PORT.
	if v := getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			errs = append(errs, fmt.Errorf("PORT %q must be a number from 1 to 65535", v))
		} else {
			cfg.Port = p
		}
	}

	if v := getenv("LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(strings.ToUpper(v))); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL %q must be debug, info, warn or error", v))
		}
	}

	cfg.GCPProject = getenv("GCP_PROJECT")
	cfg.AuthSecret = getenv("AUTH_SECRET")
	cfg.VertexLocation = getenv("VERTEX_LOCATION")
	cfg.VertexModel = getenv("VERTEX_MODEL")
	cfg.PlacesAPIKey = getenv("PLACES_API_KEY")
	cfg.RevenueCatWebhookSecret = getenv("REVENUECAT_WEBHOOK_SECRET")
	cfg.JobsSecret = getenv("JOBS_SECRET")
	cfg.StayPartners = getenv("STAY_PARTNERS")
	cfg.ExperiencePartners = getenv("EXPERIENCE_PARTNERS")
	switch cfg.DevPlan = getenv("DEV_PLAN"); {
	case cfg.DevPlan == "":
	case cfg.Env != "dev":
		errs = append(errs, errors.New("DEV_PLAN is only allowed when APP_ENV is dev"))
	case cfg.DevPlan != "free" && cfg.DevPlan != "plus":
		errs = append(errs, fmt.Errorf("DEV_PLAN %q must be free or plus", cfg.DevPlan))
	}
	switch {
	case cfg.AuthSecret == "" && cfg.Env != "dev":
		errs = append(errs, errors.New("AUTH_SECRET is required outside dev"))
	case cfg.AuthSecret != "" && len(cfg.AuthSecret) < minAuthSecret:
		errs = append(errs, fmt.Errorf("AUTH_SECRET must be at least %d characters", minAuthSecret))
	}

	errs = appendDuration(errs, getenv, "SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout)
	errs = appendDuration(errs, getenv, "REQUEST_TIMEOUT", &cfg.RequestTimeout)

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func appendDuration(errs []error, getenv func(string) string, name string, dst *time.Duration) []error {
	v := getenv(name)
	if v == "" {
		return errs
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return append(errs, fmt.Errorf("%s %q must be a positive duration such as 10s", name, v))
	}
	*dst = d
	return errs
}
