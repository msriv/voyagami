// Package split divides an amount of money between people in integer minor
// units. The parts always add up to the total, and a remainder is handed out
// the same way every time, so the app preview and the server agree.
//
// Who shares an expense is decided when the expense is saved: a person who
// joins or leaves the trip later is simply not in (or is in) the list of the
// expenses they were part of.
package split

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
)

// Mode says how the values in Part are read.
type Mode string

const (
	// Equal ignores Value; everyone pays the same, give or take one minor unit.
	Equal Mode = "equal"
	// Exact reads Value as the person's amount in minor units. They must add up to the total.
	Exact Mode = "exact"
	// Percent reads Value as hundredths of a percent (2500 is 25%). They must add up to 10000.
	Percent Mode = "percent"
	// Shares reads Value as a weight (2 shares pay twice what 1 share pays). At least one must be above zero.
	Shares Mode = "shares"
)

// BasisPoints is 100% in Percent mode.
const BasisPoints = 10000

// Part is one person in a split.
type Part struct {
	UID   string
	Value int64
}

// Allocation is what one person pays.
type Allocation struct {
	UID         string
	AmountMinor int64
}

// Split divides total between parts and returns one allocation per part, in
// the same order as parts. A negative total (a refund) is split the same way
// and every amount is negative.
//
// Remainders go by the largest-remainder method: each person gets the whole
// minor units of their exact share, then the leftover units go one each to the
// people with the biggest fractions, ties broken by UID. The result for a
// person does not depend on the order of parts.
func Split(total int64, mode Mode, parts []Part) ([]Allocation, error) {
	if len(parts) == 0 {
		return nil, errors.New("split: at least one person is needed")
	}
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		if p.UID == "" {
			return nil, errors.New("split: every person needs an ID")
		}
		if seen[p.UID] {
			return nil, fmt.Errorf("split: %s is listed twice", p.UID)
		}
		seen[p.UID] = true
	}
	if total == minInt64 {
		return nil, errors.New("split: amount is out of range")
	}
	sign := int64(1)
	abs := total
	if total < 0 {
		sign, abs = -1, -total
	}

	var out []Allocation
	var err error
	switch mode {
	case Equal:
		w := make([]int64, len(parts))
		for i := range w {
			w[i] = 1
		}
		out, err = proportional(abs, parts, w, int64(len(parts)))
	case Shares:
		out, err = byWeight(abs, parts, "shares", 0)
	case Percent:
		out, err = byWeight(abs, parts, "percentages", BasisPoints)
	case Exact:
		out, err = exact(abs, parts)
	default:
		return nil, fmt.Errorf("split: unknown mode %q", mode)
	}
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].AmountMinor *= sign
	}
	return out, nil
}

const minInt64 = -1 << 63

// byWeight reads each Value as a weight. If wantSum is above zero the weights must add up to it.
func byWeight(total int64, parts []Part, what string, wantSum int64) ([]Allocation, error) {
	w := make([]int64, len(parts))
	var sum int64
	for i, p := range parts {
		if p.Value < 0 {
			return nil, fmt.Errorf("split: %s cannot be negative", what)
		}
		if sum > (1<<62)-p.Value {
			return nil, fmt.Errorf("split: %s are too large", what)
		}
		w[i] = p.Value
		sum += p.Value
	}
	switch {
	case wantSum > 0 && sum != wantSum:
		return nil, fmt.Errorf("split: %s must add up to %d, not %d", what, wantSum, sum)
	case sum == 0:
		return nil, fmt.Errorf("split: at least one of the %s must be above zero", what)
	}
	return proportional(total, parts, w, sum)
}

func exact(total int64, parts []Part) ([]Allocation, error) {
	out := make([]Allocation, len(parts))
	var sum int64
	for i, p := range parts {
		if p.Value < 0 {
			return nil, errors.New("split: amounts cannot be negative")
		}
		if sum > total {
			break // already over; reported below
		}
		sum += p.Value
		out[i] = Allocation{UID: p.UID, AmountMinor: p.Value}
	}
	if sum != total {
		return nil, fmt.Errorf("split: amounts add up to %d, not %d", sum, total)
	}
	return out, nil
}

// proportional gives person i total*w[i]/sumW, rounded by largest remainder.
func proportional(total int64, parts []Part, w []int64, sumW int64) ([]Allocation, error) {
	type slot struct {
		idx  int
		rem  *big.Int
		base int64
	}
	slots := make([]slot, len(parts))
	var given int64
	den := big.NewInt(sumW)
	for i := range parts {
		num := new(big.Int).Mul(big.NewInt(total), big.NewInt(w[i]))
		q, r := new(big.Int).QuoRem(num, den, new(big.Int))
		slots[i] = slot{idx: i, rem: r, base: q.Int64()}
		given += slots[i].base
	}
	left := total - given // fewer than len(parts) units
	order := make([]int, len(slots))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		sa, sb := slots[order[a]], slots[order[b]]
		if c := sa.rem.Cmp(sb.rem); c != 0 {
			return c > 0
		}
		return parts[sa.idx].UID < parts[sb.idx].UID
	})
	for _, o := range order {
		if left == 0 {
			break
		}
		// A zero weight has a zero fraction and sorts last, so it only gets a unit if nobody else can.
		slots[o].base++
		left--
	}
	out := make([]Allocation, len(parts))
	for i, s := range slots {
		out[i] = Allocation{UID: parts[i].UID, AmountMinor: s.base}
	}
	return out, nil
}
