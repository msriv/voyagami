// Package devlog is the dev-only sink for app crash and error logs, so a
// developer can read what a phone reported without the phone attached.
package devlog

import (
	"context"
	"time"
)

// Entry is one report from the app.
type Entry struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"deviceId"`
	AppVersion  string    `json:"appVersion"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Stack       string    `json:"stack,omitempty"`
	Route       string    `json:"route,omitempty"`
	Breadcrumbs []string  `json:"breadcrumbs,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Store keeps entries.
type Store interface {
	Add(ctx context.Context, e Entry) error
	// List returns entries newest first. An empty deviceID means every device.
	List(ctx context.Context, deviceID string, since time.Time, limit int) ([]Entry, error)
}

// Limits on what one report may hold.
const (
	MaxBodyBytes   = 32 << 10
	MaxMessage     = 2000
	MaxStack       = 8000
	MaxBreadcrumbs = 30
	MaxCrumb       = 200
	MaxShort       = 200
)

// Levels the app may report.
var Levels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true, "fatal": true}

func cut(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// Clean trims every field to its limit.
func Clean(e Entry) Entry {
	e.DeviceID = cut(e.DeviceID, MaxShort)
	e.AppVersion = cut(e.AppVersion, MaxShort)
	e.Message = cut(e.Message, MaxMessage)
	e.Stack = cut(e.Stack, MaxStack)
	e.Route = cut(e.Route, MaxShort)
	if len(e.Breadcrumbs) > MaxBreadcrumbs {
		e.Breadcrumbs = e.Breadcrumbs[len(e.Breadcrumbs)-MaxBreadcrumbs:]
	}
	for i, b := range e.Breadcrumbs {
		e.Breadcrumbs[i] = cut(b, MaxCrumb)
	}
	return e
}
