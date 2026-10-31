// Package notify sends push notifications. A Notifier talks to one push
// service; Service finds a person's devices and cleans up dead tokens.
package notify

import (
	"context"
	"errors"
	"log/slog"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Message is what a device shows. Data carries a "kind" and ids for the deep
// link; it never carries free text from people.
type Message struct {
	Title string
	Body  string
	Data  map[string]string
}

// ErrInvalidToken means the push service no longer knows the token (the app
// was removed or the token was replaced). The device is deleted.
var ErrInvalidToken = errors.New("push token is no longer valid")

// Notifier sends one message to one device.
type Notifier interface {
	Send(ctx context.Context, d model.Device, m Message) error
}

// Service sends to all of a person's devices.
type Service struct {
	Devices repo.Devices
	Sender  Notifier
	Logger  *slog.Logger
}

// SendToUser sends the message to every device the person has and returns how
// many were accepted. Devices whose token is invalid are removed. One bad
// device does not stop the others; the first other error is returned.
func (s *Service) SendToUser(ctx context.Context, uid string, m Message) (int, error) {
	devices, err := s.Devices.List(ctx, uid)
	if err != nil {
		return 0, err
	}
	sent := 0
	var first error
	for _, d := range devices {
		err := s.Sender.Send(ctx, d, m)
		switch {
		case err == nil:
			sent++
		case errors.Is(err, ErrInvalidToken):
			if derr := s.Devices.Delete(ctx, uid, d.ID); derr != nil && first == nil {
				first = derr
			}
		case first == nil:
			first = err
		}
	}
	return sent, first
}
