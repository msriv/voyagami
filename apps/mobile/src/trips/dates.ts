// Calendar dates travel as "YYYY-MM-DD" strings, never as Dates with a time zone attached.

export const toIso = (y: number, m: number, d: number) =>
  `${String(y).padStart(4, "0")}-${String(m + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;

export const isoToDate = (iso: string) => {
  const [y = 0, m = 1, d = 1] = iso.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d));
};

export const daysBetween = (start: string, end: string) =>
  Math.round((isoToDate(end).getTime() - isoToDate(start).getTime()) / 86_400_000) + 1;

export interface DateRange {
  start?: string;
  end?: string;
}

/** First tap sets the start, the next later (or same) day sets the end, anything else starts again. */
export function pickDay(range: DateRange, day: string): DateRange {
  if (!range.start || range.end || day < range.start) return { start: day };
  return { start: range.start, end: day };
}

/** The cells of a month: blanks before day 1 (so the first day lands under its weekday), then every day. */
export function monthCells(year: number, month: number, firstWeekday: number): (string | null)[] {
  const lead = (new Date(Date.UTC(year, month, 1)).getUTCDay() - firstWeekday + 7) % 7;
  const count = new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
  return [
    ...Array<null>(lead).fill(null),
    ...Array.from({ length: count }, (_, i) => toIso(year, month, i + 1)),
  ];
}

/** Every day from start to end inclusive, as ISO strings. Stops at `max` so a typo cannot build a huge list. */
export function daysInRange(start: string, end: string, max = 60): string[] {
  const n = Math.min(daysBetween(start, end), max);
  const first = isoToDate(start).getTime();
  return Array.from({ length: n }, (_, i) =>
    new Date(first + i * 86_400_000).toISOString().slice(0, 10),
  );
}
