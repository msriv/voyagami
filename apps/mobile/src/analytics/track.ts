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
export function track(name: string, props?: EventProps): void {
  if (!enabled) return;
  waiting.push({ name, at: new Date().toISOString(), ...(props ? { props } : {}) });
  if (waiting.length > MAX_WAITING) waiting = waiting.slice(-MAX_WAITING);
  wake.forEach((w) => w());
}

export function setTrackingEnabled(on: boolean): void {
  enabled = on;
  if (!on) waiting = [];
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
