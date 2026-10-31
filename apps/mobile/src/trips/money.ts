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

export type RateRatio = { numerator: number; denominator: number };

function gcd(a: number, b: number): number {
  return b === 0 ? a : gcd(b, a % b);
}

/**
 * Reads a typed rate ("how many TO for one FROM", e.g. "25000" for VND per USD) into the minor-unit
 * ratio the API stores. Undefined for junk, zero, or numbers too long to hold exactly.
 */
export function rateFromMajor(text: string, from: string, to: string): RateRatio | undefined {
  const m = /^(\d+)(?:[.,](\d{0,6}))?$/.exec(text.replace(/\s/g, ""));
  if (!m) return undefined;
  const fraction = m[2] ?? "";
  const scaled = Number(m[1] + fraction);
  if (!Number.isSafeInteger(scaled) || scaled < 1) return undefined;
  let numerator = scaled * 10 ** minorDigits(to);
  let denominator = 10 ** fraction.length * 10 ** minorDigits(from);
  if (!Number.isSafeInteger(numerator) || !Number.isSafeInteger(denominator)) return undefined;
  const g = gcd(numerator, denominator);
  numerator /= g;
  denominator /= g;
  return { numerator, denominator };
}

/** The rate as "TO per one FROM" in major units, for showing and editing. */
export function rateToMajor(r: RateRatio, from: string, to: string): number {
  return (r.numerator / r.denominator) * 10 ** (minorDigits(from) - minorDigits(to));
}

/** Original minor units to settlement minor units, rounded half away from zero like the server. */
export function convertMinor(minor: number, r: RateRatio): number {
  return Math.round((minor * r.numerator) / r.denominator);
}
