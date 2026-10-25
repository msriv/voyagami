/**
 * Splits an amount of money between people in integer minor units.
 *
 * This is the same algorithm as services/api/internal/split, and both are tested against
 * vectors.json, so the preview in the app matches what the server saves. The server is the
 * source of truth.
 */

export type SplitMode = "equal" | "exact" | "percent" | "shares";

/** 100% in percent mode, in hundredths of a percent. */
export const BASIS_POINTS = 10000;

export interface SplitPart {
  uid: string;
  /** Ignored for equal; a weight for shares; hundredths of a percent for percent; minor units for exact. */
  value: number;
}

export interface Allocation {
  uid: string;
  amountMinor: number;
}

export class SplitError extends Error {}

/**
 * Remainders go by the largest-remainder method: everyone gets the whole minor units of their
 * exact share, then the leftover units go one each to the biggest fractions, ties by UID. A
 * person's amount does not depend on the order of `parts`. A negative total (a refund) is split
 * the same way with every amount negative.
 */
export function split(totalMinor: number, mode: SplitMode, parts: SplitPart[]): Allocation[] {
  if (!Number.isSafeInteger(totalMinor))
    throw new SplitError("amount must be a whole number of minor units");
  if (parts.length === 0) throw new SplitError("at least one person is needed");
  const seen = new Set<string>();
  for (const p of parts) {
    if (p.uid === "") throw new SplitError("every person needs an ID");
    if (seen.has(p.uid)) throw new SplitError(`${p.uid} is listed twice`);
    seen.add(p.uid);
    if (!Number.isSafeInteger(p.value)) throw new SplitError("values must be whole numbers");
  }

  const sign = totalMinor < 0 ? -1n : 1n;
  const total = BigInt(totalMinor) * sign;
  let amounts: bigint[];
  switch (mode) {
    case "equal":
      amounts = proportional(
        total,
        parts,
        parts.map(() => 1n),
      );
      break;
    case "shares":
      amounts = proportional(total, parts, weights(parts, "shares", 0n));
      break;
    case "percent":
      amounts = proportional(total, parts, weights(parts, "percentages", BigInt(BASIS_POINTS)));
      break;
    case "exact":
      amounts = exact(total, parts);
      break;
    default:
      throw new SplitError(`unknown mode ${String(mode)}`);
  }
  return parts.map((p, i) => ({ uid: p.uid, amountMinor: Number((amounts[i] ?? 0n) * sign) }));
}

function weights(parts: SplitPart[], what: string, wantSum: bigint): bigint[] {
  const w = parts.map((p) => {
    if (p.value < 0) throw new SplitError(`${what} cannot be negative`);
    return BigInt(p.value);
  });
  const sum = w.reduce((a, b) => a + b, 0n);
  if (wantSum > 0n && sum !== wantSum)
    throw new SplitError(`${what} must add up to ${wantSum}, not ${sum}`);
  if (sum === 0n) throw new SplitError(`at least one of the ${what} must be above zero`);
  return w;
}

function exact(total: bigint, parts: SplitPart[]): bigint[] {
  const amounts = parts.map((p) => {
    if (p.value < 0) throw new SplitError("amounts cannot be negative");
    return BigInt(p.value);
  });
  const sum = amounts.reduce((a, b) => a + b, 0n);
  if (sum !== total) throw new SplitError(`amounts add up to ${sum}, not ${total}`);
  return amounts;
}

function proportional(total: bigint, parts: SplitPart[], w: bigint[]): bigint[] {
  const den = w.reduce((a, b) => a + b, 0n);
  const slots = parts.map((_, i) => {
    const num = total * (w[i] ?? 0n);
    return { idx: i, base: num / den, rem: num % den };
  });
  let left = total - slots.reduce((a, s) => a + s.base, 0n);
  const order = [...slots].sort((a, b) => {
    if (a.rem !== b.rem) return a.rem > b.rem ? -1 : 1;
    const ua = parts[a.idx]?.uid ?? "";
    const ub = parts[b.idx]?.uid ?? "";
    return ua < ub ? -1 : ua > ub ? 1 : 0;
  });
  for (const s of order) {
    if (left === 0n) break;
    s.base += 1n;
    left -= 1n;
  }
  return slots.map((s) => s.base);
}
