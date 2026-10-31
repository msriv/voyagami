package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestConsentsRoundTrip(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewConsents(c)
	uid := model.NewID()
	if got, err := s.Get(ctx, uid); err != nil || got.Analytics != model.ConsentUnset || !got.UpdatedAt.IsZero() {
		t.Fatalf("new: %+v %v", got, err)
	}
	granted, denied := model.ConsentGranted, model.ConsentDenied
	now := time.Now()
	if _, err := s.Update(ctx, uid, model.ConsentsPatch{Analytics: &granted, Photos: &denied}, now); err != nil {
		t.Fatal(err)
	}
	got, err := s.Update(ctx, uid, model.ConsentsPatch{Analytics: &denied}, now.Add(time.Minute))
	if err != nil || got.Analytics != model.ConsentDenied || got.Photos != model.ConsentDenied || got.Location != model.ConsentUnset {
		t.Fatalf("update: %+v %v", got, err)
	}
	if again, _ := s.Get(ctx, uid); again.Analytics != got.Analytics || !again.UpdatedAt.Equal(got.UpdatedAt) {
		t.Errorf("get after update: %+v, want %+v", again, got)
	}
	unset := model.ConsentUnset
	if _, err := s.Update(ctx, uid, model.ConsentsPatch{Location: &unset}, now); !model.IsValidation(err) {
		t.Errorf("unset should be invalid: %v", err)
	}
}
