package locale

import (
	"errors"
	"strings"
	"testing"
)

func TestMinorDigits(t *testing.T) {
	for code, want := range map[string]int{"INR": 2, "USD": 2, "EUR": 2, "JPY": 0, "VND": 0, "KWD": 3} {
		got, err := MinorDigits(code)
		if err != nil || got != want {
			t.Errorf("MinorDigits(%s) = %d, %v; want %d", code, got, err, want)
		}
	}
	if _, err := MinorDigits("ZZZ9"); !errors.Is(err, ErrCurrency) {
		t.Errorf("unknown code error = %v, want ErrCurrency", err)
	}
}

// Symbols and spacing come from the platform data, so the tests check the parts that matter:
// the digits, the grouping and the decimals.
func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name   string
		minor  int64
		code   string
		tag    string
		digits string
	}{
		{"rupees group the Indian way", 16_000_000, "INR", "en-IN", "1,60,000.00"},
		{"dollars", 109_500, "USD", "en-US", "1,095.00"},
		{"euros the German way", 109_500, "EUR", "de-DE", "1.095,00"},
		{"yen has no decimals", 1200, "JPY", "ja-JP", "1,200"},
		{"dong has no decimals", 450_000, "VND", "vi-VN", "450.000"},
		{"dinar has three decimals", 1500, "KWD", "en-US", "1.500"},
		{"negative", -250, "USD", "en-US", "2.50"},
		{"bad tag falls back to English", 109_500, "USD", "not a tag", "1,095.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatMoney(tt.minor, tt.code, tt.tag)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, tt.digits) {
				t.Errorf("FormatMoney = %q, want it to contain %q", got, tt.digits)
			}
			if strings.Contains(got, tt.digits+"0") {
				t.Errorf("FormatMoney = %q has extra decimals", got)
			}
		})
	}
}

func TestFormatMoneyRefusesBadInput(t *testing.T) {
	if _, err := FormatMoney(1, "nope", "en"); !errors.Is(err, ErrCurrency) {
		t.Errorf("bad currency error = %v", err)
	}
	if _, err := FormatMoney(1<<60, "USD", "en"); err == nil {
		t.Error("expected an error for an amount too large to show exactly")
	}
}
