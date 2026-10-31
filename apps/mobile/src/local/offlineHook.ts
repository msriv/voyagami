import type { ApiClient, OfflineHook } from "@voyagami/api-client";

import { enqueue, flush, type Outcome, type QueueEntry } from "./queue";

/** Set on the stand-in answer so the queue's own resend is not taken for a real success. */
export const queuedHeader = "x-voyagami-queued";

/** A random ID for a record made on the phone (the idempotency key). */
export function newId(): string {
  const h = () =>
    Math.floor(Math.random() * 0x10000)
      .toString(16)
      .padStart(4, "0");
  const variant = (8 + Math.floor(Math.random() * 4)).toString(16);
  return `${h()}${h()}-${h()}-4${h().slice(1)}-${variant}${h().slice(1)}-${h()}${h()}${h()}`;
}

/**
 * When a new expense cannot reach the server, keep it in the write queue and answer as if it was
 * created (201 with what was sent), marked as queued. Every other call fails as before.
 */
export const offlineHook: OfflineHook = {
  async onNetworkError(request, e) {
    console.log("OFFLINE-HOOK", request.method, request.url, String(e));
    const m = /^\/v1\/trips\/([^/]+)\/expenses$/.exec(new URL(request.url).pathname);
    if (request.method !== "POST" || !m) return undefined;
    const body = (await request.json().catch(() => undefined)) as { id?: unknown } | undefined;
    if (!body || typeof body.id !== "string" || !body.id) return undefined;
    await enqueue({
      op: "createExpense",
      tripId: decodeURIComponent(m[1] as string),
      id: body.id,
      body: body as never,
    });
    return new Response(JSON.stringify(body), {
      status: 201,
      headers: { "Content-Type": "application/json", [queuedHeader]: "1" },
    });
  },
};

/** Sends one queued write through the normal client (so tokens refresh as usual). */
export async function sendQueued(api: ApiClient, e: QueueEntry): Promise<Outcome> {
  const { response, error } = await api.POST("/v1/trips/{tripId}/expenses", {
    params: { path: { tripId: e.tripId } },
    body: e.body,
  });
  if (response.headers.get(queuedHeader)) return "retry";
  if (response.ok) return "sent";
  const s = response.status;
  if (s === 401 || s === 408 || s === 429 || s >= 500) return "retry";
  return { reject: (error as { message?: string } | undefined)?.message ?? `Error ${s}` };
}

export const flushQueue = (api: ApiClient) => flush((e) => sendQueued(api, e));
