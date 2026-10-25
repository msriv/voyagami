// Package bestorder suggests a shorter visiting order for one day's stops.
// Stops with a fixed time are pinned and keep their place in the day; the
// others are rearranged around them. The travel times come from a Distances
// source, so straight-line estimates can later be swapped for real directions.
package bestorder

import (
	"context"
	"errors"
	"fmt"
)

// Stop is one stop of the day, in its current order.
type Stop struct {
	ID string
	// Pinned stops have a fixed time and keep their position in the day.
	Pinned bool
}

// Distances gives travel times in minutes between stops.
type Distances interface {
	// Matrix returns m where m[i][j] is the minutes from stop ids[i] to stop
	// ids[j]. It may be asymmetric. Values must not be negative.
	Matrix(ctx context.Context, ids []string) ([][]float64, error)
}

// Result is a suggested order and what it saves.
type Result struct {
	// Order lists the stop IDs in the suggested order.
	Order []string
	// BeforeMin and AfterMin are the total travel minutes of the current and suggested orders.
	BeforeMin float64
	AfterMin  float64
	// Changed is false when the current order is already the best found.
	Changed bool
}

// exactLimit is the most free stops solved by trying every order (8! = 40320).
const exactLimit = 8

// Suggest returns a visiting order that is never longer than the current one.
// It is exact for up to eight movable stops and a good local search beyond that.
// It is deterministic: the same input gives the same answer.
func Suggest(ctx context.Context, stops []Stop, src Distances) (Result, error) {
	ids := make([]string, len(stops))
	for i, s := range stops {
		ids[i] = s.ID
	}
	m, err := src.Matrix(ctx, ids)
	if err != nil {
		return Result{}, err
	}
	if err := checkMatrix(m, len(stops)); err != nil {
		return Result{}, err
	}

	var free []int // positions in the day that can be rearranged
	for i, s := range stops {
		if !s.Pinned {
			free = append(free, i)
		}
	}
	current := identity(len(stops))
	before := cost(m, current)

	best, bestCost := current, before
	if len(free) > 1 {
		cand := search(m, current, free)
		if c := cost(m, cand); c < bestCost-epsilon {
			best, bestCost = cand, c
		}
	}
	out := Result{Order: make([]string, len(best)), BeforeMin: before, AfterMin: bestCost}
	for i, idx := range best {
		out.Order[i] = stops[idx].ID
	}
	out.Changed = bestCost < before-epsilon
	return out, nil
}

// epsilon keeps rounding noise from counting as a saving.
const epsilon = 1e-9

func checkMatrix(m [][]float64, n int) error {
	if len(m) != n {
		return fmt.Errorf("bestorder: matrix has %d rows for %d stops", len(m), n)
	}
	for i, row := range m {
		if len(row) != n {
			return fmt.Errorf("bestorder: matrix row %d has %d columns for %d stops", i, len(row), n)
		}
		for _, v := range row {
			if v < 0 || v != v {
				return errors.New("bestorder: travel times must be zero or more")
			}
		}
	}
	return nil
}

func identity(n int) []int {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return p
}

// cost is the total travel time along order (an open path).
func cost(m [][]float64, order []int) float64 {
	var t float64
	for i := 1; i < len(order); i++ {
		t += m[order[i-1]][order[i]]
	}
	return t
}

// search returns the best order found with pinned stops fixed. free lists the
// positions that may hold any of the free stops.
func search(m [][]float64, start, free []int) []int {
	if len(free) <= exactLimit {
		return exact(m, start, free)
	}
	return improve(m, nearest(m, start, free), free)
}

// exact tries every arrangement of the free stops in the free positions.
func exact(m [][]float64, start, free []int) []int {
	best := append([]int(nil), start...)
	bestCost := cost(m, best)
	cur := append([]int(nil), start...)
	var walk func(k int)
	walk = func(k int) {
		if k == len(free) {
			if c := cost(m, cur); c < bestCost-epsilon {
				bestCost = c
				copy(best, cur)
			}
			return
		}
		for i := k; i < len(free); i++ {
			cur[free[k]], cur[free[i]] = cur[free[i]], cur[free[k]]
			walk(k + 1)
			cur[free[k]], cur[free[i]] = cur[free[i]], cur[free[k]]
		}
	}
	walk(0)
	return best
}

// nearest fills each free position with the closest unused free stop to the
// stop before it. The first free position keeps its current stop when it is the
// start of the day.
func nearest(m [][]float64, start, free []int) []int {
	out := append([]int(nil), start...)
	pool := make([]int, len(free))
	for i, p := range free {
		pool[i] = start[p]
	}
	for _, pos := range free {
		pick := 0
		if pos > 0 {
			from := out[pos-1]
			for i := range pool {
				if m[from][pool[i]] < m[from][pool[pick]] {
					pick = i
				}
			}
		}
		out[pos] = pool[pick]
		pool = append(pool[:pick], pool[pick+1:]...)
	}
	return out
}

// improve applies swaps and reversals of free positions while they shorten the path.
func improve(m [][]float64, order, free []int) []int {
	cur := append([]int(nil), order...)
	curCost := cost(m, cur)
	for improved := true; improved; {
		improved = false
		for a := 0; a < len(free); a++ {
			for b := a + 1; b < len(free); b++ {
				for _, try := range []func([]int){
					func(o []int) { o[free[a]], o[free[b]] = o[free[b]], o[free[a]] },
					func(o []int) { reverse(o, free, a, b) },
				} {
					cand := append([]int(nil), cur...)
					try(cand)
					if c := cost(m, cand); c < curCost-epsilon {
						cur, curCost, improved = cand, c, true
					}
				}
			}
		}
	}
	return cur
}

// reverse reverses the stops held by free[a..b].
func reverse(o, free []int, a, b int) {
	for i, j := a, b; i < j; i, j = i+1, j-1 {
		o[free[i]], o[free[j]] = o[free[j]], o[free[i]]
	}
}
