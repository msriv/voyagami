// Package locale is the Go side of the locale module (packages/locale holds the app's). It knows
// currency decimals from ISO 4217 data and formats money for a language tag. Nothing here names
// a currency symbol or a country.
package locale

import (
	"errors"
	"fmt"
	"math"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// ErrCurrency means the code is not an ISO 4217 currency.
var ErrCurrency = errors.New("locale: unknown currency code")

// maxExactMinor is the largest amount a float64 shows exactly.
const maxExactMinor = 1 << 53

func parse(code string) (currency.Unit, error) {
	unit, err := currency.ParseISO(code)
	if err != nil {
		return currency.Unit{}, fmt.Errorf("%w: %q", ErrCurrency, code)
	}
	return unit, nil
}

// MinorDigits returns the number of decimal places in an ISO 4217 currency's minor unit
// (2 for USD and INR, 0 for JPY and VND, 3 for KWD).
func MinorDigits(code string) (int, error) {
	unit, err := parse(code)
	if err != nil {
		return 0, err
	}
	scale, _ := currency.Standard.Rounding(unit)
	return scale, nil
}

// FormatMoney formats an amount held as integer minor units for a BCP 47 language tag such as
// "en-IN". An unknown or malformed tag falls back to English.
//
// shortcut: display only, so amounts above 2^53 minor units are refused; use a decimal type if one is ever needed.
func FormatMoney(amountMinor int64, code, tag string) (string, error) {
	unit, err := parse(code)
	if err != nil {
		return "", err
	}
	if amountMinor > maxExactMinor || amountMinor < -maxExactMinor {
		return "", errors.New("locale: amount too large to format exactly")
	}
	scale, _ := currency.Standard.Rounding(unit)
	value := float64(amountMinor) / math.Pow10(scale)
	lang, err := language.Parse(tag)
	if err != nil {
		lang = language.English
	}
	return message.NewPrinter(lang).Sprint(currency.NarrowSymbol(unit.Amount(value))), nil
}
