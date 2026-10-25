package bestorder

import (
	"context"
	"math"
	"slices"
	"strconv"
	"testing"
)

// line puts stop i at position pos[i] on a straight road, one minute per unit.
type line map[string]float64

func (l line) Matrix(_ context.Context, ids []string) ([][]float64, error) {
	m := make([][]float64, len(ids))
	for i, a := range ids {
		m[i] = make([]float64, len(ids))
		for j, b := range ids {
			m[i][j] = math.Abs(l[a] - l[b])
		}
	}
	return m, nil
}

func stops(ids ...string) []Stop {
	out := make([]Stop, len(ids))
	for i, id := range ids {
		out[i] = Stop{ID: id}
	}
	return out
}

func mustSuggest(t *testing.T, s []Stop, src Distances) Result {
	t.Helper()
	r, err := Suggest(context.Background(), s, src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestUntanglesAZigZag(t *testing.T) {
	src := line{"a": 0, "b": 10, "c": 1, "d": 9, "e": 2}
	r := mustSuggest(t, stops("a", "b", "c", "d", "e"), src)
	if !r.Changed || r.BeforeMin != 10+9+8+7 || r.AfterMin != 10 {
		t.Fatalf("got %+v", r)
	}
	if !slices.Equal(r.Order, []string{"a", "c", "e", "d", "b"}) {
		t.Fatalf("order %v", r.Order)
	}
}

func TestAlreadyBestKeepsOrder(t *testing.T) {
	src := line{"a": 0, "b": 1, "c": 2, "d": 3}
	r := mustSuggest(t, stops("a", "b", "c", "d"), src)
	if r.Changed || !slices.Equal(r.Order, []string{"a", "b", "c", "d"}) || r.BeforeMin != r.AfterMin {
		t.Fatalf("got %+v", r)
	}
}

func TestEqualCostKeepsCurrentOrder(t *testing.T) {
	// Every trip costs the same, so nothing is better and nothing moves.
	flat := flatDistances(1)
	r := mustSuggest(t, stops("x", "y", "z", "w"), flat)
	if r.Changed || !slices.Equal(r.Order, []string{"x", "y", "z", "w"}) {
		t.Fatalf("got %+v", r)
	}
}

type flatDistances float64

func (f flatDistances) Matrix(_ context.Context, ids []string) ([][]float64, error) {
	m := make([][]float64, len(ids))
	for i := range m {
		m[i] = make([]float64, len(ids))
		for j := range m[i] {
			if i != j {
				m[i][j] = float64(f)
			}
		}
	}
	return m, nil
}

func TestPinnedStopsKeepTheirPlace(t *testing.T) {
	// "lunch" is at a fixed time in slot 2 and must not move, even though a
	// shorter path would visit it last.
	src := line{"a": 0, "b": 10, "lunch": 5, "c": 1, "d": 9}
	in := []Stop{{ID: "a"}, {ID: "b"}, {ID: "lunch", Pinned: true}, {ID: "c"}, {ID: "d"}}
	r := mustSuggest(t, in, src)
	if r.Order[2] != "lunch" {
		t.Fatalf("pinned stop moved: %v", r.Order)
	}
	if r.AfterMin > r.BeforeMin {
		t.Fatalf("got longer: %+v", r)
	}
	if !slices.Equal(sorted(r.Order), sorted([]string{"a", "b", "lunch", "c", "d"})) {
		t.Fatalf("stops lost or added: %v", r.Order)
	}
}

func TestAllPinnedOrOneStopChangesNothing(t *testing.T) {
	src := line{"a": 0, "b": 9, "c": 1}
	pinned := []Stop{{ID: "a", Pinned: true}, {ID: "b", Pinned: true}, {ID: "c", Pinned: true}}
	if r := mustSuggest(t, pinned, src); r.Changed {
		t.Fatalf("all pinned: %+v", r)
	}
	for _, s := range [][]Stop{nil, stops("a")} {
		if r := mustSuggest(t, s, src); r.Changed || len(r.Order) != len(s) || r.AfterMin != 0 {
			t.Fatalf("%d stops: %+v", len(s), r)
		}
	}
}

// Beyond the exact limit the search must still never be worse, keep every stop,
// and fix an obvious tangle.
func TestManyStopsImproveAndAreDeterministic(t *testing.T) {
	src := line{}
	var ids []string
	for i := 0; i < 14; i++ {
		id := "s" + strconv.Itoa(i)
		ids = append(ids, id)
		// Interleave the near and far ends of the road.
		if i%2 == 0 {
			src[id] = float64(i / 2)
		} else {
			src[id] = float64(100 - i/2)
		}
	}
	in := stops(ids...)
	first := mustSuggest(t, in, src)
	if first.AfterMin >= first.BeforeMin || !first.Changed {
		t.Fatalf("no improvement: %+v", first)
	}
	if !slices.Equal(sorted(first.Order), sorted(ids)) {
		t.Fatalf("stops lost or added: %v", first.Order)
	}
	// 14 stops on a line: the best open path covers the span once.
	if first.AfterMin > 100+30 {
		t.Fatalf("still far from short: %v", first.AfterMin)
	}
	if again := mustSuggest(t, in, src); !slices.Equal(again.Order, first.Order) {
		t.Fatalf("not deterministic: %v vs %v", again.Order, first.Order)
	}
}

func TestAsymmetricTimesAreRespected(t *testing.T) {
	// One-way streets: going from higher to lower numbers is quick, the other way is slow.
	m := matrixSource{{0, 50, 50}, {1, 0, 50}, {50, 1, 0}}
	r := mustSuggest(t, stops("x0", "x1", "x2"), m)
	if !slices.Equal(r.Order, []string{"x2", "x1", "x0"}) || r.AfterMin != 2 {
		t.Fatalf("got %+v", r)
	}
}

type matrixSource [][]float64

func (m matrixSource) Matrix(context.Context, []string) ([][]float64, error) { return m, nil }

func TestBadMatrixIsAnError(t *testing.T) {
	for name, m := range map[string]matrixSource{
		"wrong rows":   {{0, 1}},
		"ragged":       {{0, 1}, {1}},
		"negative":     {{0, -1}, {1, 0}},
		"not a number": {{0, math.NaN()}, {1, 0}},
	} {
		if _, err := Suggest(context.Background(), stops("a", "b"), m); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestStraightEstimate(t *testing.T) {
	// Hanoi Old Quarter to Ho Chi Minh Mausoleum is about 2.7 km, so about 8 minutes at 20 km/h.
	s := Straight{KmPerHour: 20, Points: map[string]Point{
		"quarter": {21.0340, 105.8500}, "mausoleum": {21.0369, 105.8348}, "far": {21.0, 105.0},
	}}
	m, err := s.Matrix(context.Background(), []string{"quarter", "mausoleum"})
	if err != nil {
		t.Fatal(err)
	}
	if m[0][0] != 0 || m[0][1] != m[1][0] || m[0][1] < 4 || m[0][1] > 7 {
		t.Fatalf("matrix %v", m)
	}
	if _, err := s.Matrix(context.Background(), []string{"quarter", "nowhere"}); err == nil {
		t.Error("unknown stop should fail")
	}
	if _, err := (Straight{Points: s.Points}).Matrix(context.Background(), []string{"quarter"}); err == nil {
		t.Error("zero speed should fail")
	}
}

// The exact search matches a plain brute force over every order.
func TestExactMatchesBruteForce(t *testing.T) {
	src := line{"a": 3, "b": 17, "c": 4, "d": 12, "e": 1, "f": 8}
	ids := []string{"a", "b", "c", "d", "e", "f"}
	m, _ := src.Matrix(context.Background(), ids)
	want := math.Inf(1)
	permute(identity(len(ids)), 0, func(p []int) { want = math.Min(want, cost(m, p)) })
	r := mustSuggest(t, stops(ids...), src)
	if r.AfterMin != want {
		t.Fatalf("got %v, brute force %v", r.AfterMin, want)
	}
}

func permute(p []int, k int, visit func([]int)) {
	if k == len(p) {
		visit(p)
		return
	}
	for i := k; i < len(p); i++ {
		p[k], p[i] = p[i], p[k]
		permute(p, k+1, visit)
		p[k], p[i] = p[i], p[k]
	}
}

func sorted(s []string) []string {
	c := slices.Clone(s)
	slices.Sort(c)
	return c
}
