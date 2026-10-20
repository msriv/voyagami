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
}

const (
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
