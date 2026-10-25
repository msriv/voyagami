import { minorDigits } from "@voyagami/locale";

/**
 * Reads what someone typed as an amount, in whole minor units of the currency. Accepts "." or ","
 * as the decimal mark and ignores spaces. Returns undefined for zero, negatives, junk, or more
 * decimals than the currency has.
 * shortcut: no thousands separators ("1,234.50" is refused), until the amount field is a keypad.
 */
export function parseAmount(text: string, currency: string): number | undefined {
  const digits = minorDigits(currency);
  const m = /^(\d+)(?:[.,](\d*))?$/.exec(text.replace(/\s/g, ""));
  if (!m) return undefined;
  const fraction = m[2] ?? "";
  if (fraction.length > digits) return undefined;
  const minor = Number(m[1] + fraction.padEnd(digits, "0"));
  return Number.isSafeInteger(minor) && minor > 0 ? minor : undefined;
}

/** Today's calendar date in a time zone, as YYYY-MM-DD. */
export function todayIn(timeZone: string, now = new Date()): string {
  // Not for display: en-CA is only used because it writes YYYY-MM-DD.
  // eslint-disable-next-line no-restricted-properties
  return new Intl.DateTimeFormat("en-CA", { timeZone }).format(now);
}
