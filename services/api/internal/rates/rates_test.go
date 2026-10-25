package rates

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFixedQuote(t *testing.T) {
	f := NewFixed()
	ctx := context.Background()

	// 1 USD = 25,000 VND; USD has 2 decimals and VND has 0, so 100 cents -> 25,000 dong.
	q, err := f.Quote(ctx, "usd", "VND")
	if err != nil || q.Rate.Numerator != 250 || q.Rate.Denominator != 1 || q.From != "USD" {
		t.Fatalf("USD>VND: %+v, %v", q, err)
	}
	// 1,000 yen -> 1,000 x 1/150 USD = 6.67 USD = 667 cents (half away from zero is the expense rule).
	q, err = f.Quote(ctx, "JPY", "USD")
	if err != nil || q.Rate.Numerator != 2 || q.Rate.Denominator != 3 {
		t.Fatalf("JPY>USD: %+v, %v", q, err)
	}
	// Same currency is 1:1.
	if q, err = f.Quote(ctx, "INR", "INR"); err != nil || q.Rate.Numerator != 1 || q.Rate.Denominator != 1 {
		t.Fatalf("INR>INR: %+v, %v", q, err)
	}
	// Going there and back gives the inverse.
	a, _ := f.Quote(ctx, "EUR", "INR")
	b, _ := f.Quote(ctx, "INR", "EUR")
	if a.Rate.Numerator != b.Rate.Denominator || a.Rate.Denominator != b.Rate.Numerator {
		t.Errorf("not inverse: %+v %+v", a.Rate, b.Rate)
	}
	if _, err := f.Quote(ctx, "USD", "XXX"); !errors.Is(err, ErrNoRate) {
		t.Errorf("unknown: %v", err)
	}
}

type counting struct{ n int }

func (c *counting) Quote(ctx context.Context, from, to string) (Quote, error) {
	c.n++
	return NewFixed().Quote(ctx, from, to)
}

func TestCachedKeepsAQuoteForTheDay(t *testing.T) {
	src := &counting{}
	c := NewCached(src)
	clock := time.Date(2026, 10, 25, 23, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return clock }
	ctx := context.Background()

	for range 3 {
		if _, err := c.Quote(ctx, "usd", "VND"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.Quote(ctx, "USD", "THB"); err != nil || src.n != 2 {
		t.Fatalf("source calls: %d, %v", src.n, err)
	}
	if _, err := c.Quote(ctx, "USD", "XXX"); !errors.Is(err, ErrNoRate) {
		t.Errorf("errors are not cached: %v", err)
	}
	clock = clock.Add(2 * time.Hour) // next UTC day
	if _, err := c.Quote(ctx, "USD", "VND"); err != nil || src.n != 4 {
		t.Errorf("after midnight: %d calls, %v", src.n, err)
	}
}
