package analytics

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"regexp"
	"time"
)

// Context is what the app says about itself for a batch, and what the server
// knows about the person. Every value is a short identifier or empty.
type Context struct {
	SessionID   string
	Platform    string
	OSVersion   string
	AppVersion  string
	AppLanguage string
	Plan        string
	HomeRegion  string
}

// Record is one row of the BigQuery table analytics.events. The JSON names are
// the column names (infra/modules/analytics); times are microseconds since the
// epoch, which BigQuery reads as TIMESTAMP.
type Record struct {
	EventID      string `json:"eventId"`
	Name         string `json:"name"`
	EventTime    int64  `json:"eventTime"`
	ReceivedTime int64  `json:"receivedTime"`
	UID          string `json:"uid,omitempty"`
	SessionID    string `json:"sessionId,omitempty"`
	Platform     string `json:"platform,omitempty"`
	OSVersion    string `json:"osVersion,omitempty"`
	AppVersion   string `json:"appVersion,omitempty"`
	AppLanguage  string `json:"appLanguage,omitempty"`
	Plan         string `json:"plan,omitempty"`
	HomeRegion   string `json:"homeRegion,omitempty"`
	Source       string `json:"source"`
	Props        string `json:"props,omitempty"` // JSON object as text
}

// Sources of an event.
const (
	SourceApp    = "app"
	SourceServer = "server"
)

// Context values allow "+" too, for versions such as 1.2.0+45.
var contextRe = regexp.MustCompile(`^[A-Za-z0-9_.:+-]{1,64}$`)

func cleanContext(c Context) Context {
	for _, v := range []*string{&c.SessionID, &c.Platform, &c.OSVersion, &c.AppVersion, &c.AppLanguage, &c.Plan, &c.HomeRegion} {
		if !contextRe.MatchString(*v) {
			*v = ""
		}
	}
	return c
}

// Records turns checked events into table rows. uid may be empty: the event
// is then counted without saying whose it is, and carries no properties or
// context that could tie it to a person.
func Records(events []Event, uid string, c Context, source string, now time.Time) []Record {
	c = cleanContext(c)
	out := make([]Record, 0, len(events))
	for _, e := range events {
		r := Record{
			EventID: newID(), Name: e.Name, EventTime: e.At.UnixMicro(), ReceivedTime: now.UnixMicro(),
			Source: source,
		}
		if uid != "" {
			r.UID, r.SessionID, r.Platform, r.OSVersion, r.AppVersion, r.AppLanguage, r.Plan, r.HomeRegion =
				uid, c.SessionID, c.Platform, c.OSVersion, c.AppVersion, c.AppLanguage, c.Plan, c.HomeRegion
			if len(e.Props) > 0 {
				b, _ := json.Marshal(e.Props) // values are strings, finite numbers and bools
				r.Props = string(b)
			}
		}
		out = append(out, r)
	}
	return out
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Sink receives rows. Production publishes to Pub/Sub (PubSub); local dev logs.
type Sink interface {
	Add(ctx context.Context, rows []Record) error
}

// LogSink logs how many rows arrived and nothing about them. For local dev,
// where there is no topic.
type LogSink struct{ Logger *slog.Logger }

// Add logs the count.
func (s LogSink) Add(_ context.Context, rows []Record) error {
	s.Logger.Info("analytics events (not stored: no ANALYTICS_TOPIC)", "count", len(rows))
	return nil
}
