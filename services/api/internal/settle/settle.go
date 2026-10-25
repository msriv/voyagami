// Package settle works out who owes whom on a trip. Everything is integer
// minor units in the trip's settlement currency.
package settle

import "sort"

// Spend is what one expense does to balances: who paid and each person's share.
type Spend struct {
	PaidBy string
	Total  int64
	Shares map[string]int64
}

// Payment is money already handed over outside the app.
type Payment struct {
	From, To string
	Amount   int64
}

// Transfer is one suggested payment.
type Transfer struct {
	From, To string
	Amount   int64
}

// Balances returns each person's net position. Positive means the group owes
// them, negative means they owe the group. The values always add up to zero
// as long as every spend's shares add up to its total.
func Balances(spends []Spend, payments []Payment) map[string]int64 {
	b := map[string]int64{}
	for _, s := range spends {
		b[s.PaidBy] += s.Total
		for uid, a := range s.Shares {
			b[uid] -= a
		}
	}
	for _, p := range payments {
		b[p.From] += p.Amount
		b[p.To] -= p.Amount
	}
	return b
}

// Suggest returns payments that clear every balance. Each payment settles at
// least one person completely, so there are never more than n-1 of them. It
// pairs the largest debtor with the largest creditor, ties by ID, so the
// answer is the same every time. This is not always the theoretical minimum
// (that problem is NP-hard); it is exact for the usual one-payer trip.
func Suggest(balances map[string]int64) []Transfer {
	type side struct {
		uid string
		amt int64
	}
	var owed, owing []side
	for uid, a := range balances {
		switch {
		case a > 0:
			owed = append(owed, side{uid, a})
		case a < 0:
			owing = append(owing, side{uid, -a})
		}
	}
	order := func(s []side) {
		sort.Slice(s, func(i, j int) bool {
			if s[i].amt != s[j].amt {
				return s[i].amt > s[j].amt
			}
			return s[i].uid < s[j].uid
		})
	}
	var out []Transfer
	for len(owed) > 0 && len(owing) > 0 {
		order(owed)
		order(owing)
		c, d := &owed[0], &owing[0]
		amt := min(c.amt, d.amt)
		out = append(out, Transfer{From: d.uid, To: c.uid, Amount: amt})
		c.amt -= amt
		d.amt -= amt
		if c.amt == 0 {
			owed = owed[1:]
		}
		if d.amt == 0 {
			owing = owing[1:]
		}
	}
	return out
}
