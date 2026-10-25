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

export type Move = { kind: "up" } | { kind: "down" } | { kind: "day"; date: string };

/** Sorts after any real order key, so a stop moved to the end of a day shows there at once. */
const END = "￿";

/**
 * What a move sends to the server, and what the list looks like right away.
 * Returns undefined when the move does nothing (already first, last or on that day).
 */
export function planMove(
  stops: Stop[],
  stop: Stop,
  move: Move,
): { body: { beforeStopId?: string; date?: string }; local: Stop[] } | undefined {
  const day = stopsOn(stops, stop.date);
  const i = day.findIndex((s) => s.id === stop.id);
  const swapWith = (other: Stop) =>
    stops.map((s) =>
      s.id === stop.id
        ? { ...s, order: other.order }
        : s.id === other.id
          ? { ...s, order: stop.order }
          : s,
    );
  if (move.kind === "up") {
    const prev = day[i - 1];
    return prev ? { body: { beforeStopId: prev.id }, local: swapWith(prev) } : undefined;
  }
  if (move.kind === "down") {
    const next = day[i + 1];
    if (!next) return undefined;
    const after = day[i + 2];
    return {
      body: after ? { beforeStopId: after.id } : { date: stop.date },
      local: swapWith(next),
    };
  }
  if (move.date === stop.date) return undefined;
  return {
    body: { date: move.date },
    local: stops.map((s) => (s.id === stop.id ? { ...s, date: move.date, order: END } : s)),
  };
}

/**
 * Reads the two coordinate fields. Both blank gives a position of (0, 0), which
 * stands for "no pin yet"; a half-filled or out-of-range pin gives undefined.
 * shortcut: (0, 0) is stored as a real position, until the map pin and place search supply one.
 */
export function parsePin(lat: string, lng: string): { lat: number; lng: number } | undefined {
  const a = lat.trim();
  const b = lng.trim();
  if (a === "" && b === "") return { lat: 0, lng: 0 };
  const x = Number(a);
  const y = Number(b);
  if (a === "" || b === "" || !Number.isFinite(x) || !Number.isFinite(y)) return undefined;
  if (Math.abs(x) > 90 || Math.abs(y) > 180) return undefined;
  return { lat: x, lng: y };
}

/**
 * A day's stops reordered to the given IDs, shown before the server answers. The day keeps
 * its existing order keys; they are handed out again in the new sequence.
 */
export function applyOrder(stops: Stop[], date: string, ids: string[]): Stop[] {
  const keys = stopsOn(stops, date).map((s) => s.order);
  const byId = new Map(ids.map((id, i) => [id, keys[i]]));
  return stops.map((s) => {
    const order = s.date === date ? byId.get(s.id) : undefined;
    return order === undefined ? s : { ...s, order };
  });
}
