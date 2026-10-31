package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/analytics"
)

func TestEventsAdd(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewEvents(c)
	now := time.Now()
	err := s.Add(ctx, "u-events", []analytics.Event{
		{Name: "trip_created", At: now, Props: map[string]any{"destinations": 2.0, "solo": true}},
		{Name: "sign_in", At: now, Props: map[string]any{}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	docs, err := c.Collection(eventsCollection).Where("uid", "==", "u-events").Documents(ctx).GetAll()
	if err != nil || len(docs) != 2 {
		t.Fatalf("stored %d: %v", len(docs), err)
	}
}
