type Listener = (feature: string) => void;
const listeners = new Set<Listener>();

/** Tells the paywall that the server said `limit_reached` for a plan feature. */
export function emitLimitReached(feature: string): void {
  listeners.forEach((l) => l(feature));
}

export function onLimitReached(listener: Listener): () => void {
  listeners.add(listener);
  return () => void listeners.delete(listener);
}
