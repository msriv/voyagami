package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/msriv/voyagami/services/api/internal/repo"
)

// purgeBatch bounds one run; the scheduler calls daily and the rest waits for tomorrow.
const purgeBatch = 50

// accountPurge carries out the account deletions whose 30 days are over.
type accountPurge struct {
	deletions repo.Deletions
	logger    *slog.Logger
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
		p.logger.Info("account purged", "count", done) // no user id: it names a person
	}
	return done, nil
}
