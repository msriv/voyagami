// Package logging builds the JSON logger in the shape Cloud Logging expects.
//
// Logs must never carry personal data: no phone numbers, exact locations or
// free text. Log identifiers and counts, not request or response bodies.
package logging

import (
	"io"
	"log/slog"
)

// New returns a JSON logger that writes Cloud Logging fields: severity,
// message and timestamp.
func New(w io.Writer, level slog.Level, env string) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: cloudLoggingAttr,
	})
	return slog.New(h).With("env", env)
}

func cloudLoggingAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
		a.Value = slog.StringValue(severity(a.Value.Any().(slog.Level)))
	case slog.MessageKey:
		a.Key = "message"
	case slog.TimeKey:
		a.Key = "timestamp"
	}
	return a
}

func severity(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARNING"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}
