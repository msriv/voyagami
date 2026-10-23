/** Digits after the decimal point for an ISO 4217 currency, from the platform's currency data (JPY and VND are 0, KWD is 3). */
export function minorDigits(currency: string): number {
  return new Intl.NumberFormat("en", { style: "currency", currency }).resolvedOptions()
    .maximumFractionDigits as number;
}

/**
 * Formats an amount held as integer minor units (cents, paise, yen) in a currency, for a locale
 * such as "en-IN". Symbols, grouping and decimals all come from the platform. Throws on a
 * non-integer, and on an amount too large to show exactly.
 * shortcut: display only, so amounts above 2^53 minor units are refused; move to a decimal library if one is needed.
 */
export function formatMoney(minor: number, currency: string, locale: string): string {
  if (!Number.isSafeInteger(minor)) {
    throw new RangeError("Money must be a whole number of minor units");
  }
  const digits = minorDigits(currency);
  return new Intl.NumberFormat(locale, { style: "currency", currency }).format(
    minor / 10 ** digits,
  );
}
