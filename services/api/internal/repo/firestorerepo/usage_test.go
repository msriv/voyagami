package firestorerepo

import (
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestUsageAdd(t *testing.T) {
	c := testClient(t)
	trips := NewTrips(c)
	usage := NewUsage(c)
	ctx := testCtx(t)
	trip, err := trips.Create(ctx, sampleTrip("2027-02-01", "2027-02-05"), model.Member{UID: "u1", DisplayName: "Asha"})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{1, 2} {
		if err := usage.Add(ctx, trip.ID, model.UsageForwardedBookings, n); err != nil {
			t.Fatal(err)
		}
	}
	if err := usage.Add(ctx, trip.ID, model.UsageBestOrderRuns, 1); err != nil {
		t.Fatal(err)
	}
	got, err := trips.Get(ctx, trip.ID)
	if err != nil || got.Usage.ForwardedBookings != 3 || got.Usage.BestOrderRuns != 1 || got.Usage.Files != 0 {
		t.Fatalf("usage = %+v, %v", got.Usage, err)
	}
}
