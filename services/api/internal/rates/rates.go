// Package rates answers "how many X is one Y worth" so the app can suggest the
// rate for an expense in another currency. The rate itself is then stored on
// the expense, so a later change here never alters old expenses.
package rates

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/msriv/voyagami/services/api/internal/locale"
	"github.com/msriv/voyagami/services/api/internal/model"
)

// ErrNoRate means the provider has no rate for the pair.
var ErrNoRate = errors.New("rates: no rate for that pair")

// Quote converts minor units of From into minor units of To:
// to = from x Numerator / Denominator, with the rounding rule in model.Rate.
// It already accounts for each currency's decimals (JPY and VND have none).
type Quote struct {
	From, To string
	Rate     model.Rate
	AsOf     string // YYYY-MM-DD, UTC
}

// Provider looks up rates. Real sources plug in behind it (P1); dev uses Fixed.
type Provider interface {
	Quote(ctx context.Context, from, to string) (Quote, error)
}

// microPerUSD is how many millionths of each currency one US dollar buys.
// shortcut: illustrative dev numbers, not a live feed; swap in a real source behind Provider in P1.
var microPerUSD = map[string]int64{
	"USD": 1_000_000, "EUR": 920_000, "GBP": 790_000, "INR": 83_500_000, "VND": 25_000_000_000,
	"JPY": 150_000_000, "THB": 35_500_000, "SGD": 1_350_000, "IDR": 15_800_000_000,
	"AUD": 1_520_000, "CAD": 1_370_000, "AED": 3_672_500,
}

// Fixed is the dev provider: a small built-in table, crossed through USD.
type Fixed struct{ now func() time.Time }

// NewFixed returns the dev provider.
func NewFixed() *Fixed { return &Fixed{now: time.Now} }

// Quote returns the table rate for the pair.
func (f *Fixed) Quote(_ context.Context, from, to string) (Quote, error) {
	from, to = strings.ToUpper(from), strings.ToUpper(to)
	fromMicro, okFrom := microPerUSD[from]
	toMicro, okTo := microPerUSD[to]
	if !okFrom || !okTo {
		return Quote{}, ErrNoRate
	}
	fromDigits, err := locale.MinorDigits(from)
	if err != nil {
		return Quote{}, err
	}
	toDigits, err := locale.MinorDigits(to)
	if err != nil {
		return Quote{}, err
	}
	// 1 major From = toMicro/fromMicro major To; scale both sides to minor units.
	num, den := toMicro*pow10(toDigits), fromMicro*pow10(fromDigits)
	g := gcd(num, den)
	return Quote{From: from, To: to, Rate: model.Rate{Numerator: num / g, Denominator: den / g}, AsOf: f.now().UTC().Format(time.DateOnly)}, nil
}

func pow10(n int) int64 {
	p := int64(1)
	for ; n > 0; n-- {
		p *= 10
	}
	return p
}

func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Cached keeps each pair's quote for the rest of the UTC day, so a busy day
// asks the source once per pair.
type Cached struct {
	src Provider
	now func() time.Time

	mu  sync.Mutex
	day string
	m   map[string]Quote
}

// NewCached wraps a provider with the daily cache.
func NewCached(src Provider) *Cached { return &Cached{src: src, now: time.Now, m: map[string]Quote{}} }

// Quote returns today's cached quote, asking the source on the first request.
func (c *Cached) Quote(ctx context.Context, from, to string) (Quote, error) {
	from, to = strings.ToUpper(from), strings.ToUpper(to)
	day := c.now().UTC().Format(time.DateOnly)
	key := from + ">" + to
	c.mu.Lock()
	if c.day != day {
		c.day, c.m = day, map[string]Quote{}
	}
	if q, ok := c.m[key]; ok {
		c.mu.Unlock()
		return q, nil
	}
	c.mu.Unlock()
	q, err := c.src.Quote(ctx, from, to)
	if err != nil {
		return Quote{}, err
	}
	c.mu.Lock()
	if c.day == day {
		c.m[key] = q
	}
	c.mu.Unlock()
	return q, nil
}
