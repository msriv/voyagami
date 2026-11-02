package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/analytics"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type purgeFake struct {
	repo.Deletions
	due    []string
	failOn string
	purged []string
}

func (f *purgeFake) Due(context.Context, time.Time, int) ([]string, error) { return f.due, nil }
func (f *purgeFake) Purge(_ context.Context, uid string, _ time.Time) error {
	if uid == f.failOn {
		return errors.New("boom")
	}
	f.purged = append(f.purged, uid)
	return nil
}

func TestAccountPurgeRun(t *testing.T) {
	f := &purgeFake{due: []string{"a", "b", "c"}, failOn: "b"}
	p := accountPurge{deletions: f, logger: slog.New(slog.DiscardHandler)}
	n, err := p.Run(context.Background())
	if err == nil || n != 1 || len(f.purged) != 1 || f.purged[0] != "a" {
		t.Fatalf("stops at the failure: n=%d err=%v purged=%v", n, err, f.purged)
	}
	f.failOn = ""
	f.purged = nil
	if n, err := p.Run(context.Background()); err != nil || n != 3 {
		t.Errorf("retry finishes the rest: n=%d err=%v", n, err)
	}
}

type rowsFake struct{ rows []analytics.Record }

func (f *rowsFake) Add(_ context.Context, r []analytics.Record) error {
	f.rows = append(f.rows, r...)
	return nil
}

func TestAccountPurgeMarksAnalytics(t *testing.T) {
	ev := &rowsFake{}
	p := accountPurge{deletions: &purgeFake{due: []string{"u1"}}, logger: slog.New(slog.DiscardHandler), events: ev}
	if _, err := p.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(ev.rows) != 2 || ev.rows[0].Name != "account_purged" || ev.rows[0].UID != "u1" ||
		ev.rows[1].Name != "account_deleted" || ev.rows[1].UID != "" {
		t.Fatalf("want a purge marker for u1 and an anonymous count: %+v", ev.rows)
	}
}
