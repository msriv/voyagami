package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestRecapVideos(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewRecapVideos(c)
	now := time.Now()

	a, created, err := s.Enqueue(ctx, model.RecapVideo{TripID: "t1", RequestedBy: "u1", MaxSeconds: 30, Watermark: true, CreatedAt: now})
	if err != nil || !created || a.State != model.RecapQueued {
		t.Fatalf("enqueue: %+v %v %v", a, created, err)
	}
	again, created, err := s.Enqueue(ctx, model.RecapVideo{TripID: "t1", CreatedAt: now})
	if err != nil || created || again.ID != a.ID {
		t.Fatalf("second enqueue should return the active job: %+v %v %v", again, created, err)
	}
	b, _, _ := s.Enqueue(ctx, model.RecapVideo{TripID: "t2", CreatedAt: now.Add(time.Second)})

	got, ok, err := s.Claim(ctx, now, now.Add(-time.Hour))
	if err != nil || !ok || got.ID != a.ID || got.State != model.RecapRendering || got.Attempts != 1 {
		t.Fatalf("claim should take the oldest: %+v %v %v", got, ok, err)
	}
	if err := s.Progress(ctx, a.ID, 40); err != nil {
		t.Fatal(err)
	}
	if l, _ := s.Latest(ctx, "t1"); l.Progress != 40 {
		t.Errorf("progress %d", l.Progress)
	}
	if got, _, _ := s.Claim(ctx, now, now.Add(-time.Hour)); got.ID != b.ID {
		t.Errorf("a running job is not claimed again: %+v", got)
	}
	// Nothing left, unless a running job went stale.
	if _, ok, _ := s.Claim(ctx, now, now.Add(-time.Hour)); ok {
		t.Error("nothing should be left")
	}
	if got, ok, _ := s.Claim(ctx, now.Add(time.Hour), now.Add(30*time.Minute)); !ok || got.Attempts != 2 {
		t.Errorf("a stuck job is picked up again: %+v %v", got, ok)
	}

	if err := s.Fail(ctx, a.ID, "x", true); err != nil {
		t.Fatal(err)
	}
	if l, _ := s.Latest(ctx, "t1"); l.State != model.RecapQueued {
		t.Errorf("requeued: %s", l.State)
	}
	_ = s.Fail(ctx, a.ID, "We couldn't make your video.", false)
	if l, _ := s.Latest(ctx, "t1"); l.State != model.RecapFailed || l.Error == "" {
		t.Errorf("failed: %+v", l)
	}
	_ = s.Finish(ctx, b.ID, "videos/b.mp4")
	if l, _ := s.Latest(ctx, "t2"); l.State != model.RecapReady || l.Object != "videos/b.mp4" || l.Progress != 100 {
		t.Errorf("finished: %+v", l)
	}
	if _, err := s.Latest(ctx, "nobody"); err == nil {
		t.Error("unknown trip should be not found")
	}
}
