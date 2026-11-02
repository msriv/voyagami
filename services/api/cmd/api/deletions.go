package main

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/msriv/voyagami/services/api/internal/analytics"
	"github.com/msriv/voyagami/services/api/internal/config"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// purgeBatch bounds one run; the scheduler calls daily and the rest waits for tomorrow.
const purgeBatch = 50

// accountPurge carries out the account deletions whose 30 days are over.
type accountPurge struct {
	deletions repo.Deletions
	logger    *slog.Logger
	events    analytics.Sink // nil means no analytics rows to remove
	now       func() time.Time
}

// Run purges the accounts that are due and returns how many it finished. It
// stops at the first failure; the account stays due and the next run retries.
func (p accountPurge) Run(ctx context.Context) (int, error) {
	now := time.Now()
	if p.now != nil {
		now = p.now()
	}
	due, err := p.deletions.Due(ctx, now, purgeBatch)
	if err != nil {
		return 0, err
	}
	done := 0
	for _, uid := range due {
		if err := p.deletions.Purge(ctx, uid, now); err != nil {
			return done, err
		}
		done++
		p.markPurged(ctx, uid, now)
		p.logger.Info("account purged", "count", done) // no user id: it names a person
	}
	return done, nil
}

// markPurged tells analytics to remove the person's rows: the nightly purge
// query deletes every row of a uid that has an account_purged row (VOY-229).
// It is sent whatever the consent was, since rows may predate a withdrawal.
// An anonymous account_deleted keeps the count.
func (p accountPurge) markPurged(ctx context.Context, uid string, now time.Time) {
	if p.events == nil {
		return
	}
	ev := []analytics.Event{{Name: "account_purged", At: now}}
	rows := append(analytics.Records(ev, uid, analytics.Context{}, analytics.SourceServer, now),
		analytics.Records([]analytics.Event{{Name: "account_deleted", At: now}}, "", analytics.Context{}, analytics.SourceServer, now)...)
	if err := p.events.Add(ctx, rows); err != nil {
		// The account is gone either way. Its rows expire with their partitions.
		p.logger.Error("analytics purge marker failed", "error", err)
	}
}

// newAnalyticsSink publishes to ANALYTICS_TOPIC, or only logs counts without it.
func newAnalyticsSink(ctx context.Context, cfg config.Config, project string, logger *slog.Logger) (analytics.Sink, error) {
	if cfg.AnalyticsTopic == "" || project == "" || project == firestore.DetectProjectID {
		return analytics.LogSink{Logger: logger}, nil
	}
	return analytics.NewPubSub(ctx, project, cfg.AnalyticsTopic)
}
