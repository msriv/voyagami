/** Minutes a time zone is ahead of UTC at an instant. */
function offsetMinutes(timeZone: string, at: number): number {
  // Not for display: the parts are only read back as numbers.
  // eslint-disable-next-line no-restricted-properties
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    hourCycle: "h23",
    year: "numeric",
    month: "numeric",
    day: "numeric",
    hour: "numeric",
    minute: "numeric",
  }).formatToParts(at);
  const n = (type: string) => Number(parts.find((p) => p.type === type)?.value);
  const asUtc = Date.UTC(n("year"), n("month") - 1, n("day"), n("hour"), n("minute"));
  return Math.round((asUtc - Math.floor(at / 60000) * 60000) / 60000);
}

/**
 * The UTC instant (ISO string) of a wall-clock date "YYYY-MM-DD" and time "HH:mm" in a zone.
 * shortcut: a time that does not exist (the hour skipped when clocks go forward) lands an hour off.
 */
export function zonedToUtc(date: string, time: string, timeZone: string): string {
  const [y = 0, mo = 1, d = 1] = date.split("-").map(Number);
  const [h = 0, mi = 0] = time.split(":").map(Number);
  const wall = Date.UTC(y, mo - 1, d, h, mi);
  const first = wall - offsetMinutes(timeZone, wall) * 60000;
  return new Date(wall - offsetMinutes(timeZone, first) * 60000).toISOString();
}
