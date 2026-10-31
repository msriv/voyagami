package notify

import (
	"context"
	"log/slog"

	"github.com/msriv/voyagami/services/api/internal/model"
)

// Log is the dev Notifier: it sends nothing and logs that it would have. The
// log holds the kind of message and the end of the token, never the title,
// the body or the data, which can hold names and places.
type Log struct{ Logger *slog.Logger }

// Send logs a redacted line.
func (l Log) Send(_ context.Context, d model.Device, m Message) error {
	tail := d.PushToken
	if len(tail) > 4 {
		tail = tail[len(tail)-4:]
	}
	l.Logger.Info("push (not sent, dev)", "platform", d.Platform, "tokenEnd", tail, "kind", m.Data["kind"])
	return nil
}
