import type { EventArgs, EventName } from "./events";

export type EventProps = Record<string, string | number | boolean>;
export interface AnalyticsEvent {
  name: string;
  at: string;
  props?: EventProps;
}

const MAX_WAITING = 100;
let waiting: AnalyticsEvent[] = [];
let enabled = false;
const wake = new Set<() => void>();

/** Records a product event. Does nothing unless the person turned analytics on. */
export function track<K extends EventName>(name: K, ...args: EventArgs<K>): void {
  if (!enabled) return;
  const props = args[0] as EventProps | undefined;
  waiting.push({ name, at: new Date().toISOString(), ...(props ? { props } : {}) });
  if (waiting.length > MAX_WAITING) waiting = waiting.slice(-MAX_WAITING);
  wake.forEach((w) => w());
}

export function setTrackingEnabled(on: boolean): void {
  enabled = on;
  if (!on) waiting = [];
  else
    for (const [name, props] of pending.splice(0))
      track(name, ...([props] as EventArgs<typeof name>));
}

// Sign-in happens before tracking can be on (it needs a session and consent).
// These wait in memory, are sent once both are there, and are lost on restart.
let pending: [EventName, EventProps | undefined][] = [];

/** Like track, for events that happen before sign-in. */
export function trackOnceOn<K extends EventName>(name: K, ...args: EventArgs<K>): void {
  if (enabled) return track(name, ...args);
  pending = [
    ...pending,
    [name, args[0] as EventProps | undefined] as [EventName, EventProps | undefined],
  ].slice(-10);
}

/** Takes up to `max` events off the waiting list. */
export function takeEvents(max: number): AnalyticsEvent[] {
  return waiting.splice(0, max);
}

/** Puts events back at the front after a failed send. */
export function putBack(events: AnalyticsEvent[]): void {
  waiting = [...events, ...waiting].slice(-MAX_WAITING);
}

export function onTracked(fn: () => void): () => void {
  wake.add(fn);
  return () => void wake.delete(fn);
}
