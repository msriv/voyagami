package firestorerepo

import (
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/billing"
	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestBillingApply(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	b := NewBilling(c)
	acc := NewEntitlements(c)
	uid := model.NewID()
	base := time.Now().Truncate(time.Millisecond)
	ev := func(id, typ string, at time.Duration, over func(*billing.Event)) billing.Event {
		e := billing.Event{ID: id + uid, Type: typ, UID: uid, ProductID: "plus_monthly", PeriodType: "NORMAL", At: base.Add(at), ExpiresAt: base.Add(30 * 24 * time.Hour)}
		if over != nil {
			over(&e)
		}
		return e
	}
	plus := func() bool {
		p, _, err := acc.Account(ctx, uid)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	apply := func(e billing.Event) {
		t.Helper()
		if err := b.Apply(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	apply(ev("a", "INITIAL_PURCHASE", time.Second, nil))
	if !plus() {
		t.Fatal("purchase must give Plus")
	}
	apply(ev("a", "INITIAL_PURCHASE", time.Second, nil)) // replay
	// A late EXPIRATION from before the purchase must not undo it.
	apply(ev("old", "EXPIRATION", 0, nil))
	if !plus() {
		t.Fatal("an older event undid Plus")
	}
	apply(ev("x", "EXPIRATION", 2*time.Second, nil))
	if plus() {
		t.Fatal("expiry must end Plus")
	}

	// Trip Pass: purchase, refund before and after.
	trip := model.NewID()
	pass := func(id, typ string, at time.Duration) billing.Event {
		return ev(id, typ, at, func(e *billing.Event) {
			e.ProductID, e.TransactionID, e.TripID = "trip_pass_7d", "tx"+trip, trip
			if typ == "CANCELLATION" {
				e.CancelReason = "CUSTOMER_SUPPORT"
			}
		})
	}
	has := func() bool {
		ok, err := b.HasPass(ctx, trip)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}
	if has() {
		t.Fatal("no pass yet")
	}
	apply(pass("p1", "NON_RENEWING_PURCHASE", time.Second))
	if !has() {
		t.Fatal("pass must unlock the trip")
	}
	apply(pass("p2", "CANCELLATION", 2*time.Second))
	if has() {
		t.Fatal("refund must remove the pass")
	}
}
