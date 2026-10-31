package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestDevicesPutListDelete(t *testing.T) {
	s := NewDevices(testClient(t))
	ctx := testCtx(t)
	uid := model.NewID()
	d := model.Device{ID: "d1", Platform: model.PlatformIOS, PushToken: "tok-1", LastSeenAt: time.Now()}

	if err := s.Put(ctx, uid, d); err != nil {
		t.Fatal(err)
	}
	d.PushToken = "tok-2"
	if err := s.Put(ctx, uid, d); err != nil {
		t.Fatal(err)
	}
	got, err := s.List(ctx, uid)
	if err != nil || len(got) != 1 || got[0].PushToken != "tok-2" || got[0].Platform != model.PlatformIOS {
		t.Fatalf("list: %+v %v", got, err)
	}
	if err := s.Put(ctx, uid, model.Device{ID: "d2", Platform: "web", PushToken: "x"}); !model.IsValidation(err) {
		t.Fatalf("invalid device: %v", err)
	}
	for range 2 { // removing twice is fine
		if err := s.Delete(ctx, uid, "d1"); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := s.List(ctx, uid); len(got) != 0 {
		t.Fatalf("not removed: %+v", got)
	}
}
