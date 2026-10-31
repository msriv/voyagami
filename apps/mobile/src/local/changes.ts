/** A tiny "something changed" signal, so the sync badge can re-read the queue without polling. */
const listeners = new Set<() => void>();

export const onQueueChange = (fn: () => void): (() => void) => {
  listeners.add(fn);
  return () => void listeners.delete(fn);
};

export const queueChanged = () => listeners.forEach((fn) => fn());
