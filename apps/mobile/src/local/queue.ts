import type { components } from "@voyagami/api-client";

import { localStore } from "./store";

/**
 * Writes made without a connection. Each entry is one API call to send later, in order.
 *
 * Only calls the server can safely see twice go here: today that is creating an expense with an ID
 * made on the phone, which the server answers with the stored expense when it sees it again. Add an
 * op only when the server dedupes it (see docs/sync.md).
 */
export type QueuedOp = {
  op: "createExpense";
  tripId: string;
  body: components["schemas"]["ExpenseCreate"];
};

export type QueueEntry = QueuedOp & {
  /** The idempotency key: the ID the new record will have. */
  id: string;
  createdAt: number;
  attempts: number;
  /** Do not try again before this time (ms since epoch). */
  nextAt: number;
};

export type FailedEntry = QueueEntry & { reason: string };

const QUEUE = "queue";
const FAILED = "queue-failed";

export const MAX_BACKOFF_MS = 5 * 60_000;

/** 2s, 4s, 8s ... up to five minutes. */
export function backoffMs(attempts: number): number {
  return Math.min(2000 * 2 ** Math.max(0, attempts - 1), MAX_BACKOFF_MS);
}

async function read<T>(key: string): Promise<T[]> {
  return ((await localStore().get(key)) as T[] | undefined) ?? [];
}

export const pendingWrites = () => read<QueueEntry>(QUEUE);
export const failedWrites = () => read<FailedEntry>(FAILED);

/** Adds a write. The same ID twice is one entry, so a double tap cannot queue two. */
export async function enqueue(op: QueuedOp & { id: string }, now = Date.now()): Promise<void> {
  const list = await pendingWrites();
  if (list.some((e) => e.id === op.id)) return;
  await localStore().put(QUEUE, [...list, { ...op, createdAt: now, attempts: 0, nextAt: now }]);
}

export async function dismissFailed(id: string): Promise<void> {
  await localStore().put(
    FAILED,
    (await failedWrites()).filter((e) => e.id !== id),
  );
}

/** What sending an entry came to. */
export type Outcome = "sent" | "retry" | { reject: string };

/**
 * Sends due entries in order. Stops at the first one that must wait, so later writes never pass an
 * earlier one. A rejected entry (the server said no) moves to the failed list so the person can see
 * it. Returns how many were sent.
 */
export async function flush(
  send: (e: QueueEntry) => Promise<Outcome>,
  now = Date.now(),
): Promise<number> {
  let list = await pendingWrites();
  let sent = 0;
  while (list.length > 0) {
    const [head, ...rest] = list as [QueueEntry, ...QueueEntry[]];
    if (head.nextAt > now) break;
    const out = await send(head).catch((): Outcome => "retry");
    if (out === "retry") {
      const attempts = head.attempts + 1;
      await localStore().put(QUEUE, [
        { ...head, attempts, nextAt: now + backoffMs(attempts) },
        ...rest,
      ]);
      break;
    }
    list = rest;
    if (out === "sent") sent++;
    else
      await localStore().put(FAILED, [...(await failedWrites()), { ...head, reason: out.reject }]);
    await localStore().put(QUEUE, list);
  }
  return sent;
}
