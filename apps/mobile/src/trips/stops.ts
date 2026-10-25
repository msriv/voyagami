import { formatTime } from "@voyagami/locale";

import type { components } from "@voyagami/api-client";

export type Stop = components["schemas"]["Stop"];

/** Stops on one day, in plan order. The API sorts by date then order; this keeps that for a day. */
export function stopsOn(stops: Stop[], date: string): Stop[] {
  return stops.filter((s) => s.date === date).sort((a, b) => (a.order < b.order ? -1 : 1));
}

const TIME = /^([01]\d|2[0-3]):([0-5]\d)$/;

/** True for "" (no time) or a valid 24-hour "HH:mm". */
export function isValidTime(value: string): boolean {
  return value === "" || TIME.test(value);
}

/** A stop's start time in the viewer's own clock style. The stored value is the stop's local time. */
export function displayTime(startLocal: string, locale: string): string {
  const m = TIME.exec(startLocal);
  if (!m) return "";
  const at = new Date(Date.UTC(2000, 0, 1, Number(m[1]), Number(m[2])));
  return formatTime(at, locale, { timeZone: "UTC" });
}
