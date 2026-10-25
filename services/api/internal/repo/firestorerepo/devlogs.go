package firestorerepo

import (
	"context"
	"errors"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/devlog"
)

const devLogsCollection = "devLogs"

// DevLogs implements devlog.Store. Dev only.
type DevLogs struct {
	client *firestore.Client
}

var _ devlog.Store = (*DevLogs)(nil)

// NewDevLogs returns the Firestore-backed dev log store.
func NewDevLogs(client *firestore.Client) *DevLogs { return &DevLogs{client: client} }

type devLogDoc struct {
	DeviceID    string    `firestore:"deviceId"`
	AppVersion  string    `firestore:"appVersion"`
	Level       string    `firestore:"level"`
	Message     string    `firestore:"message"`
	Stack       string    `firestore:"stack,omitempty"`
	Route       string    `firestore:"route,omitempty"`
	Breadcrumbs []string  `firestore:"breadcrumbs,omitempty"`
	CreatedAt   time.Time `firestore:"createdAt"`
}

// Add stores one entry.
func (s *DevLogs) Add(ctx context.Context, e devlog.Entry) error {
	d := devLogDoc{e.DeviceID, e.AppVersion, e.Level, e.Message, e.Stack, e.Route, e.Breadcrumbs, e.CreatedAt.UTC().Truncate(time.Microsecond)}
	_, _, err := s.client.Collection(devLogsCollection).Add(ctx, d)
	return mapErr("add dev log", err)
}

// maxScan caps how many of one device's entries List reads, so no composite
// index is needed: it filters on the device and sorts here.
const maxScan = 500

// List returns entries newest first.
func (s *DevLogs) List(ctx context.Context, deviceID string, since time.Time, limit int) ([]devlog.Entry, error) {
	q := s.client.Collection(devLogsCollection).Query
	if deviceID != "" {
		q = q.Where("deviceId", "==", deviceID).Limit(maxScan)
	} else {
		q = q.OrderBy("createdAt", firestore.Desc).Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()
	var out []devlog.Entry
	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapErr("list dev logs", err)
		}
		var d devLogDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, mapErr("decode dev log", err)
		}
		if d.CreatedAt.Before(since) {
			continue
		}
		out = append(out, devlog.Entry{ID: snap.Ref.ID, DeviceID: d.DeviceID, AppVersion: d.AppVersion, Level: d.Level,
			Message: d.Message, Stack: d.Stack, Route: d.Route, Breadcrumbs: d.Breadcrumbs, CreatedAt: d.CreatedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
