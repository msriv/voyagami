package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestSupportAdd(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewSupport(c)
	id, err := s.Add(ctx, model.SupportMessage{UserID: "u1", Category: "bug", Text: "map is blank", CreatedAt: time.Now(),
		Diagnostics: model.Diagnostics{AppVersion: "1.0", OS: "android"}})
	if err != nil || id == "" {
		t.Fatalf("add: %q %v", id, err)
	}
	snap, err := c.Collection(supportCollection).Doc(id).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var d supportDoc
	if err := snap.DataTo(&d); err != nil || d.UserID != "u1" || d.Text != "map is blank" || d.Diagnostics.OS != "android" {
		t.Errorf("stored %+v %v", d, err)
	}
	if _, err := s.Add(ctx, model.SupportMessage{UserID: "u1", Category: "bug"}); !model.IsValidation(err) {
		t.Errorf("empty text should be invalid: %v", err)
	}
}
