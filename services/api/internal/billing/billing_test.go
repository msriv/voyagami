package billing

import (
	"strconv"
	"testing"
	"time"
)

func body(typ, id string, ts, exp int64, extra string) []byte {
	b := `{"api_version":"1.0","event":{"id":"` + id + `","type":"` + typ + `","app_user_id":"u1","product_id":"plus_monthly",` +
		`"transaction_id":"tx1","period_type":"NORMAL","event_timestamp_ms":` + strconv.FormatInt(ts, 10) + `,"expiration_at_ms":` + strconv.FormatInt(exp, 10) + extra + `}}`
	return []byte(b)
}

func mustParse(t *testing.T, b []byte) Event {
	t.Helper()
	e, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestParse(t *testing.T) {
	e := mustParse(t, body("NON_RENEWING_PURCHASE", "e1", 1000, 0,
		`,"product_id":"trip_pass_7d","subscriber_attributes":{"tripId":{"value":"t9"}}`))
	if e.TripID != "t9" || !e.IsTripPass() || !e.ExpiresAt.IsZero() || e.UID != "u1" {
		t.Fatalf("%+v", e)
	}
	if _, err := Parse([]byte(`{"event":{}}`)); err == nil {
		t.Fatal("want error for an event with no id")
	}
}

func TestSubscriptionLifecycle(t *testing.T) {
	day := int64(24 * time.Hour / time.Millisecond)
	steps := []struct {
		name  string
		ev    []byte
		check func(Plus) bool
	}{
		{"trial starts", body("INITIAL_PURCHASE", "a", 1000, 1000+7*day, `,"period_type":"TRIAL"`),
			func(p Plus) bool { return !p.Active && !p.TrialEndsAt.IsZero() }},
		{"first paid renewal", body("RENEWAL", "b", 2000, 2000+30*day, ``),
			func(p Plus) bool { return p.Active && p.TrialEndsAt.IsZero() && p.WillRenew }},
		{"cancel keeps access", body("CANCELLATION", "c", 3000, 2000+30*day, `,"cancel_reason":"UNSUBSCRIBE"`),
			func(p Plus) bool { return p.Active && !p.WillRenew }},
		{"billing issue", body("BILLING_ISSUE", "d", 4000, 2000+30*day, ``),
			func(p Plus) bool { return p.Active && p.BillingIssue }},
		{"expires", body("EXPIRATION", "e", 5000, 2000+30*day, ``),
			func(p Plus) bool { return !p.Active }},
		{"buys again", body("INITIAL_PURCHASE", "f", 6000, 6000+30*day, ``),
			func(p Plus) bool { return p.Active && !p.BillingIssue }},
		{"refund ends it at once", body("CANCELLATION", "g", 7000, 6000+30*day, `,"cancel_reason":"CUSTOMER_SUPPORT"`),
			func(p Plus) bool { return !p.Active }},
	}
	var p Plus
	for _, s := range steps {
		var changed bool
		p, changed = Next(p, mustParse(t, s.ev))
		if !changed || !s.check(p) {
			t.Fatalf("%s: changed=%v state=%+v", s.name, changed, p)
		}
	}
}

func TestReplayAndOutOfOrder(t *testing.T) {
	renew := mustParse(t, body("RENEWAL", "r", 5000, 9000, ``))
	p, _ := Next(Plus{}, renew)
	if _, changed := Next(p, renew); changed {
		t.Fatal("a replay must change nothing")
	}
	late := mustParse(t, body("EXPIRATION", "x", 4000, 4000, ``))
	if got, changed := Next(p, late); changed || !got.Active {
		t.Fatal("an older event must not undo a newer state")
	}
}

func TestIgnoredTypes(t *testing.T) {
	if _, changed := Next(Plus{}, mustParse(t, body("TEST", "t", 1000, 0, ``))); changed {
		t.Fatal("TEST must change nothing")
	}
}
