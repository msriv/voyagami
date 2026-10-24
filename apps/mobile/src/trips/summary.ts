import type { components } from "@voyagami/api-client";

import { daysBetween, isoToDate } from "./dates";

export type Trip = components["schemas"]["Trip"];

/** Today as "YYYY-MM-DD" on the device. */
export const todayIso = (now = new Date()) =>
  `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;

/** Past means archived, or fixed dates that have ended. Everything else is upcoming, in the order the API gave. */
export function partitionTrips(trips: Trip[], today: string) {
  const upcoming: Trip[] = [];
  const past: Trip[] = [];
  for (const trip of trips) {
    const ended = trip.dates.end !== undefined && trip.dates.end < today;
    (trip.status === "past" || ended ? past : upcoming).push(trip);
  }
  return { upcoming, past };
}

export type Countdown = { kind: "ongoing" } | { kind: "in"; days: number } | { kind: "none" };

/** How far away a trip is. Trips with flexible dates have no countdown. */
export function countdown(trip: Trip, today: string): Countdown {
  const { start, end } = trip.dates;
  if (!start || !end) return { kind: "none" };
  if (today >= start && today <= end) return { kind: "ongoing" };
  return today < start ? { kind: "in", days: daysBetween(today, start) - 1 } : { kind: "none" };
}

export { isoToDate };
