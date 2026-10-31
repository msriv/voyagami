package recapvideo

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/notify"
)

type fakeStore struct {
	queue    []model.RecapVideo
	progress map[string]int
	finished map[string]string
	failed   map[string]string
	requeued []string
}

func (f *fakeStore) Enqueue(context.Context, model.RecapVideo) (model.RecapVideo, bool, error) {
	return model.RecapVideo{}, false, nil
}
func (f *fakeStore) Latest(context.Context, string) (model.RecapVideo, error) {
	return model.RecapVideo{}, nil
}
func (f *fakeStore) Claim(context.Context, time.Time, time.Time) (model.RecapVideo, bool, error) {
	if len(f.queue) == 0 {
		return model.RecapVideo{}, false, nil
	}
	j := f.queue[0]
	f.queue = f.queue[1:]
	j.Attempts++
	return j, true, nil
}
func (f *fakeStore) Progress(_ context.Context, id string, p int) error {
	f.progress[id] = p
	return nil
}
func (f *fakeStore) Finish(_ context.Context, id, o string) error { f.finished[id] = o; return nil }
func (f *fakeStore) Fail(_ context.Context, id, m string, requeue bool) error {
	if requeue {
		f.requeued = append(f.requeued, id)
		return nil
	}
	f.failed[id] = m
	return nil
}

type fakeRenderer struct {
	failIDs map[string]bool
	hang    bool
}

func (r fakeRenderer) Render(ctx context.Context, j model.RecapVideo, progress func(int)) (string, error) {
	if r.hang {
		<-ctx.Done()
		return "", ctx.Err()
	}
	progress(50)
	if r.failIDs[j.ID] {
		return "", errors.New("encoder crashed")
	}
	return "videos/" + j.ID + ".mp4", nil
}
func (fakeRenderer) URL(context.Context, string) (string, error) { return "", nil }

type fakePush struct{ got []string }

func (p *fakePush) SendToUser(_ context.Context, uid string, _ notify.Message) (int, error) {
	p.got = append(p.got, uid)
	return 1, nil
}

func newStore(jobs ...model.RecapVideo) *fakeStore {
	return &fakeStore{queue: jobs, progress: map[string]int{}, finished: map[string]string{}, failed: map[string]string{}}
}

func TestRunRendersAndNotifies(t *testing.T) {
	st := newStore(model.RecapVideo{ID: "a", RequestedBy: "u1"}, model.RecapVideo{ID: "b", RequestedBy: "u2"})
	push := &fakePush{}
	w := Worker{Store: st, Renderer: fakeRenderer{failIDs: map[string]bool{"b": true}}, Push: push, Logger: slog.New(slog.DiscardHandler)}
	n, err := w.Run(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("done %d, err %v", n, err)
	}
	if st.finished["a"] != "videos/a.mp4" || st.progress["a"] != 50 || len(push.got) != 1 || push.got[0] != "u1" {
		t.Errorf("finished %v progress %v push %v", st.finished, st.progress, push.got)
	}
	if len(st.requeued) != 1 || st.requeued[0] != "b" {
		t.Errorf("a first failure is retried: %v", st.requeued)
	}
}

func TestRunGivesUpAfterMaxAttempts(t *testing.T) {
	st := newStore(model.RecapVideo{ID: "c", Attempts: 2})
	w := Worker{Store: st, Renderer: fakeRenderer{failIDs: map[string]bool{"c": true}}, Logger: slog.New(slog.DiscardHandler)}
	if _, err := w.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if st.failed["c"] == "" || len(st.requeued) != 0 {
		t.Errorf("third failure should end the job: failed %v requeued %v", st.failed, st.requeued)
	}
}

func TestRunHonoursTimeout(t *testing.T) {
	st := newStore(model.RecapVideo{ID: "d"})
	w := Worker{Store: st, Renderer: fakeRenderer{hang: true}, Timeout: 20 * time.Millisecond, Logger: slog.New(slog.DiscardHandler)}
	start := time.Now()
	if _, err := w.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second || len(st.requeued) != 1 {
		t.Errorf("hung render should time out and retry: %v, requeued %v", time.Since(start), st.requeued)
	}
}
