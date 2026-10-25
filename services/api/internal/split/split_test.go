package split

import (
	"encoding/json"
	"math/rand"
	"os"
	"slices"
	"testing"
)

func parts(vals ...int64) []Part {
	out := make([]Part, len(vals))
	for i, v := range vals {
		out[i] = Part{UID: string(rune('a' + i)), Value: v}
	}
	return out
}

func amounts(t *testing.T, total int64, mode Mode, p []Part) []int64 {
	t.Helper()
	got, err := Split(total, mode, p)
	if err != nil {
		t.Fatalf("Split(%d, %s): %v", total, mode, err)
	}
	out := make([]int64, len(got))
	var sum int64
	for i, a := range got {
		if a.UID != p[i].UID {
			t.Fatalf("order changed: %v", got)
		}
		out[i] = a.AmountMinor
		sum += a.AmountMinor
	}
	if sum != total {
		t.Fatalf("parts add up to %d, want %d: %v", sum, total, out)
	}
	return out
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name  string
		total int64
		n     int
		want  []int64
	}{
		{"even", 900, 3, []int64{300, 300, 300}},
		{"one over goes to the first by ID", 1000, 3, []int64{334, 333, 333}},
		{"two over", 1001, 3, []int64{334, 334, 333}},
		{"fewer units than people", 2, 5, []int64{1, 1, 0, 0, 0}},
		{"zero", 0, 3, []int64{0, 0, 0}},
		{"one person", 777, 1, []int64{777}},
		{"zero-decimal currency amount", 10001, 4, []int64{2501, 2500, 2500, 2500}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := amounts(t, tt.total, Equal, parts(make([]int64, tt.n)...)); !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSharesAndPercent(t *testing.T) {
	if got := amounts(t, 1000, Shares, parts(2, 1, 1)); !slices.Equal(got, []int64{500, 250, 250}) {
		t.Errorf("shares: %v", got)
	}
	// 100 split 1:1:1 leaves one unit; every fraction ties, so the lowest ID gets it.
	if got := amounts(t, 100, Shares, parts(1, 1, 1)); !slices.Equal(got, []int64{34, 33, 33}) {
		t.Errorf("tie: %v", got)
	}
	// 3333 and 6667 basis points of 1001: exact 333.6633 and 667.3367.
	if got := amounts(t, 1001, Percent, parts(3333, 6667)); !slices.Equal(got, []int64{334, 667}) {
		t.Errorf("percent: %v", got)
	}
	if got := amounts(t, 500, Shares, parts(0, 3, 2)); !slices.Equal(got, []int64{0, 300, 200}) {
		t.Errorf("zero share pays nothing: %v", got)
	}
	if got := amounts(t, 1000, Percent, parts(10000, 0)); !slices.Equal(got, []int64{1000, 0}) {
		t.Errorf("all to one: %v", got)
	}
}

func TestExact(t *testing.T) {
	if got := amounts(t, 1000, Exact, parts(250, 750)); !slices.Equal(got, []int64{250, 750}) {
		t.Errorf("exact: %v", got)
	}
}

func TestRefundsMirrorPayments(t *testing.T) {
	if got := amounts(t, -1000, Equal, parts(0, 0, 0)); !slices.Equal(got, []int64{-334, -333, -333}) {
		t.Errorf("refund: %v", got)
	}
}

func TestBadInput(t *testing.T) {
	tests := []struct {
		name  string
		total int64
		mode  Mode
		p     []Part
	}{
		{"nobody", 100, Equal, nil},
		{"unknown mode", 100, Mode("thirds"), parts(1, 1)},
		{"empty ID", 100, Equal, []Part{{UID: ""}}},
		{"duplicate person", 100, Equal, []Part{{UID: "a"}, {UID: "a"}}},
		{"exact too little", 100, Exact, parts(10, 20)},
		{"exact too much", 100, Exact, parts(60, 60)},
		{"exact negative", 100, Exact, parts(-10, 110)},
		{"percent short", 100, Percent, parts(5000, 4000)},
		{"percent over", 100, Percent, parts(6000, 6000)},
		{"shares all zero", 100, Shares, parts(0, 0)},
		{"shares negative", 100, Shares, parts(-1, 2)},
		{"shares overflow", 100, Shares, parts(1<<62, 1<<62)},
		{"most negative amount", -1 << 63, Equal, parts(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Split(tt.total, tt.mode, tt.p); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

// Properties over many random splits, including large amounts.
func TestProperties(t *testing.T) {
	r := rand.New(rand.NewSource(7)) //nolint:gosec // seeded so failures repeat
	for range 3000 {
		n := 1 + r.Intn(9)
		total := r.Int63n(1_000_000_000_000) // up to 10 billion major units in cents
		if r.Intn(4) == 0 {
			total = r.Int63n(50)
		}
		p := make([]Part, n)
		var wsum int64
		for i := range p {
			p[i] = Part{UID: string(rune('a' + i)), Value: 1 + r.Int63n(20)}
			wsum += p[i].Value
		}
		got, err := Split(total, Shares, p)
		if err != nil {
			t.Fatal(err)
		}
		var sum int64
		for i, a := range got {
			sum += a.AmountMinor
			// Never more than one minor unit away from the exact share.
			exactLow := total * p[i].Value / wsum
			if a.AmountMinor < exactLow || a.AmountMinor > exactLow+1 {
				t.Fatalf("total %d weights %v: %v is not within one unit", total, p, got)
			}
		}
		if sum != total {
			t.Fatalf("total %d weights %v sums to %d", total, p, sum)
		}

		// The same people in another order get the same amounts.
		shuffled := slices.Clone(p)
		r.Shuffle(n, func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		again, _ := Split(total, Shares, shuffled)
		byUID := map[string]int64{}
		for _, a := range again {
			byUID[a.UID] = a.AmountMinor
		}
		for _, a := range got {
			if byUID[a.UID] != a.AmountMinor {
				t.Fatalf("order changed the answer for %s", a.UID)
			}
		}
	}
}

// TestVectors runs the cases shared with the TypeScript package.
func TestVectors(t *testing.T) {
	raw, err := os.ReadFile("../../../../packages/split/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Cases []struct {
			Name  string          `json:"name"`
			Total int64           `json:"total"`
			Mode  Mode            `json:"mode"`
			Parts [][]interface{} `json:"parts"`
			Want  []int64         `json:"want"`
			Error bool            `json:"error"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			ps := make([]Part, len(c.Parts))
			for i, p := range c.Parts {
				ps[i] = Part{UID: p[0].(string), Value: int64(p[1].(float64))}
			}
			got, err := Split(c.Total, c.Mode, ps)
			if c.Error {
				if err == nil {
					t.Fatal("want an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for i, a := range got {
				if a.AmountMinor != c.Want[i] {
					t.Fatalf("got %v, want %v", got, c.Want)
				}
			}
		})
	}
}
