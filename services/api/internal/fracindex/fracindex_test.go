package fracindex

import (
	"math/rand"
	"slices"
	"testing"
)

func between(t *testing.T, a, b string) string {
	t.Helper()
	k, err := Between(a, b)
	if err != nil {
		t.Fatalf("Between(%q, %q): %v", a, b, err)
	}
	if k <= a || (b != "" && k >= b) {
		t.Fatalf("Between(%q, %q) = %q, not strictly between", a, b, k)
	}
	if !valid(k) {
		t.Fatalf("Between(%q, %q) = %q, ends in the lowest digit", a, b, k)
	}
	return k
}

func TestBetweenCases(t *testing.T) {
	for _, c := range [][2]string{
		{"", ""}, {"V", ""}, {"", "V"}, {"A", "B"}, {"A", "C"}, {"A", "A1"}, {"A1", "A2"},
		{"Az", "B"}, {"zz", ""}, {"", "1"}, {"V", "V1"}, {"V", "V2"},
	} {
		between(t, c[0], c[1])
	}
}

func TestBetweenRejectsBadInput(t *testing.T) {
	for _, c := range [][2]string{{"B", "A"}, {"A", "A"}, {"A0", "B"}, {"a-", ""}} {
		if _, err := Between(c[0], c[1]); err == nil {
			t.Errorf("Between(%q, %q) should fail", c[0], c[1])
		}
	}
}

// Repeated inserts at the end, the front and random spots keep the order.
func TestManyInsertsStaySorted(t *testing.T) {
	r := rand.New(rand.NewSource(1)) //nolint:gosec // a seeded generator keeps the test repeatable
	keys := []string{between(t, "", "")}
	for range 2000 {
		i := r.Intn(len(keys) + 1)
		var lo, hi string
		if i > 0 {
			lo = keys[i-1]
		}
		if i < len(keys) {
			hi = keys[i]
		}
		keys = slices.Insert(keys, i, between(t, lo, hi))
	}
	if !slices.IsSorted(keys) {
		t.Fatal("keys are out of order")
	}
	if len(slices.Compact(slices.Clone(keys))) != len(keys) {
		t.Fatal("duplicate keys")
	}
}
