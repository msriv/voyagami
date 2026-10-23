export type Clock = "12h" | "24h";

interface Options {
  /** IANA zone such as "Asia/Ho_Chi_Minh". Defaults to the device's. A trip uses its own zone. */
  timeZone?: string;
}

/** A calendar date in the viewer's format, for example "21 Nov" or "Nov 21". Add the year with `withYear`. */
export function formatDate(
  date: Date | number,
  locale: string,
  { timeZone, withYear = false }: Options & { withYear?: boolean } = {},
): string {
  return new Intl.DateTimeFormat(locale, {
    day: "numeric",
    month: "short",
    ...(withYear ? { year: "numeric" } : null),
    ...(timeZone ? { timeZone } : null),
  }).format(date);
}

/** The viewer's own clock setting for a locale, used when the device does not say. */
export function defaultClock(locale: string): Clock {
  const cycle = new Intl.DateTimeFormat(locale, { hour: "numeric" }).resolvedOptions().hourCycle;
  return cycle === "h23" || cycle === "h24" ? "24h" : "12h";
}

/** A time of day, 12 or 24 hour as the viewer prefers. */
export function formatTime(
  date: Date | number,
  locale: string,
  { timeZone, clock = defaultClock(locale) }: Options & { clock?: Clock } = {},
): string {
  return new Intl.DateTimeFormat(locale, {
    hour: "numeric",
    minute: "2-digit",
    hour12: clock === "12h",
    ...(timeZone ? { timeZone } : null),
  }).format(date);
}
