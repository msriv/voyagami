// Package recapvideo runs the queue of recap video renders. The renderer
// itself (frames, music, encoding) sits behind an interface; this package owns
// the lifecycle: claim, time limit, retries, progress and the "ready" push.
package recapvideo

import (
	"context"
	"log/slog"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/notify"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// Renderer makes the video and says where to download it.
type Renderer interface {
	// Render makes the video for the job, calling progress with 0 to 100, and
	// returns the stored object's name. It must honour ctx, which carries the
	// time limit, and must not exceed job.MaxSeconds or skip the watermark.
	Render(ctx context.Context, job model.RecapVideo, progress func(percent int)) (object string, err error)
	// URL returns a short-lived download link for a rendered object.
	URL(ctx context.Context, object string) (string, error)
}

// Pusher sends a push to a person.
type Pusher interface {
	SendToUser(ctx context.Context, uid string, m notify.Message) (int, error)
}

// Worker renders queued jobs.
type Worker struct {
	Store    repo.RecapVideos
	Renderer Renderer
	Push     Pusher
	Logger   *slog.Logger
	// Timeout is the hard limit for one attempt. Default 10 minutes.
	Timeout time.Duration
	// MaxAttempts is how many tries a job gets before it fails. Default 3.
	MaxAttempts int
	// PerRun is how many jobs one run takes. Default 3.
	PerRun int
	Now    func() time.Time
}

const failedMessage = "We couldn't make your video this time. Try again later."

// Run renders queued jobs and returns how many finished. A scheduler calls it
// every minute or two. A job left rendering past the time limit (the process
// died) is picked up again.
func (w *Worker) Run(ctx context.Context) (int, error) {
	timeout, maxAttempts, perRun := w.Timeout, w.MaxAttempts, w.PerRun
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	if perRun == 0 {
		perRun = 3
	}
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	done := 0
	for i := 0; i < perRun; i++ {
		t := now()
		job, ok, err := w.Store.Claim(ctx, t, t.Add(-2*timeout))
		if err != nil {
			return done, err
		}
		if !ok {
			break
		}
		start := time.Now()
		rctx, cancel := context.WithTimeout(ctx, timeout)
		object, rerr := w.Renderer.Render(rctx, job, func(p int) { _ = w.Store.Progress(ctx, job.ID, p) })
		cancel()
		// Cost proxy: wall time per video and attempt. No trip content is logged.
		w.Logger.Info("recap video attempt", "job", job.ID, "attempt", job.Attempts, "ms", time.Since(start).Milliseconds(), "failed", rerr != nil)
		if rerr != nil {
			w.Logger.Error("recap video failed", "job", job.ID, "attempt", job.Attempts, "error", rerr)
			if err := w.Store.Fail(ctx, job.ID, failedMessage, job.Attempts < maxAttempts); err != nil {
				return done, err
			}
			continue
		}
		if err := w.Store.Finish(ctx, job.ID, object); err != nil {
			return done, err
		}
		done++
		if w.Push != nil {
			msg := notify.Message{Title: "Your trip video is ready", Body: "Open it to share.", Data: map[string]string{"type": "recap_video", "tripId": job.TripID}}
			if _, err := w.Push.SendToUser(ctx, job.RequestedBy, msg); err != nil {
				w.Logger.Warn("recap video push failed", "job", job.ID, "error", err)
			}
		}
	}
	return done, nil
}
