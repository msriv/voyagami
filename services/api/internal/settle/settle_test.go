package settle

import (
	"math/rand"
	"testing"
)

func TestOnePayerTrip(t *testing.T) {
	spends := []Spend{{PaidBy: "a", Total: 900, Shares: map[string]int64{"a": 300, "b": 300, "c": 300}}}
	b := Balances(spends, nil)
	if b["a"] != 600 || b["b"] != -300 || b["c"] != -300 {
		t.Fatalf("balances: %v", b)
	}
	got := Suggest(b)
	if len(got) != 2 || got[0] != (Transfer{"b", "a", 300}) || got[1] != (Transfer{"c", "a", 300}) {
		t.Fatalf("transfers: %v", got)
	}
}

func TestPaymentsReduceBalances(t *testing.T) {
	spends := []Spend{{PaidBy: "a", Total: 900, Shares: map[string]int64{"a": 300, "b": 300, "c": 300}}}
	b := Balances(spends, []Payment{{From: "b", To: "a", Amount: 300}})
	if b["a"] != 300 || b["b"] != 0 || b["c"] != -300 {
		t.Fatalf("balances: %v", b)
	}
	if got := Suggest(b); len(got) != 1 || got[0] != (Transfer{"c", "a", 300}) {
		t.Fatalf("transfers: %v", got)
	}
}

func TestNothingOwed(t *testing.T) {
	if got := Suggest(Balances(nil, nil)); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

// Random trips: balances add up to zero, and paying the suggestions clears everyone.
func TestProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(7)) //nolint:gosec // seeded test data, not security
	for i := 0; i < 2000; i++ {
		n := 2 + rng.Intn(7)
		uids := make([]string, n)
		for j := range uids {
			uids[j] = string(rune('a' + j))
		}
		var spends []Spend
		for k := rng.Intn(8); k >= 0; k-- {
			shares := map[string]int64{}
			var total int64
			for _, u := range uids {
				if rng.Intn(3) > 0 {
					a := int64(rng.Intn(100000)) - 20000
					shares[u] = a
					total += a
				}
			}
			spends = append(spends, Spend{PaidBy: uids[rng.Intn(n)], Total: total, Shares: shares})
		}
		b := Balances(spends, nil)
		var sum int64
		for _, a := range b {
			sum += a
		}
		if sum != 0 {
			t.Fatalf("balances add up to %d", sum)
		}
		ts := Suggest(b)
		if len(ts) > n-1 {
			t.Fatalf("%d transfers for %d people", len(ts), n)
		}
		var pays []Payment
		for _, tr := range ts {
			if tr.Amount <= 0 || tr.From == tr.To {
				t.Fatalf("bad transfer %+v", tr)
			}
			pays = append(pays, Payment(tr))
		}
		for u, a := range Balances(spends, pays) {
			if a != 0 {
				t.Fatalf("%s still has %d", u, a)
			}
		}
	}
}
