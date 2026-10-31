import { createApiClient } from "@voyagami/api-client";

import { flushQueue, newId, offlineHook, queuedHeader } from "./offlineHook";
import { enqueue, failedWrites, pendingWrites } from "./queue";
import { localStore, memoryStore, setLocalStore } from "./store";

beforeEach(() => setLocalStore(memoryStore()));

const body = (id: string) => ({
  id,
  description: "Pho",
  paidByUid: "u1",
  date: "2099-11-19",
  original: { amountMinor: 100, currency: "VND" },
  split: { method: "equal" as const, parts: [{ uid: "u1" }] },
});

function client(send: (r: Request) => Response | Promise<Response>) {
  const calls: { id: unknown; ok: boolean }[] = [];
  const api = createApiClient({
    baseUrl: "https://api.test",
    getTokens: () =>
      ({ accessToken: "a", refreshToken: "r", expiresAt: Date.now() + 1e6 }) as never,
    setTokens: () => undefined,
    offline: offlineHook,
    fetch: (async (input: RequestInfo | URL) => {
      const req = input as Request;
      const sent = await req.clone().json();
      const res = await send(req);
      calls.push({ id: sent.id, ok: res.ok });
      return res;
    }) as typeof fetch,
  });
  return { api, calls };
}

const down = () => {
  throw new TypeError("Network request failed");
};
const created = () =>
  new Response(JSON.stringify({ id: "x" }), {
    status: 201,
    headers: { "Content-Type": "application/json" },
  });

describe("offline expense", () => {
  it("queues when the network is down and answers as created", async () => {
    const { api } = client(down);
    const id = newId();
    const { response } = await api.POST("/v1/trips/{tripId}/expenses", {
      params: { path: { tripId: "t1" } },
      body: body(id),
    });
    expect(response.status).toBe(201);
    expect(response.headers.get(queuedHeader)).toBe("1");
    expect((await pendingWrites()).map((e) => e.id)).toEqual([id]);
  });

  it("sends the same ID once the network is back, and empties the queue", async () => {
    let online = false;
    const { api, calls } = client(() => (online ? created() : down()));
    const id = newId();
    await api.POST("/v1/trips/{tripId}/expenses", {
      params: { path: { tripId: "t1" } },
      body: body(id),
    });
    expect(await flushQueue(api)).toBe(0); // still down: stays queued, not duplicated
    expect(await pendingWrites()).toHaveLength(1);
    online = true;
    // The retry wait is over.
    await localStore().put(
      "queue",
      (await pendingWrites()).map((e) => ({ ...e, nextAt: 0 })),
    );
    expect(await flushQueue(api)).toBe(1);
    expect(await pendingWrites()).toEqual([]);
    expect(calls.filter((c) => c.ok).map((c) => c.id)).toEqual([id]);
  });

  it("moves an expense the server refuses to the failed list", async () => {
    const { api } = client(
      () =>
        new Response(JSON.stringify({ code: "invalid_request", message: "Bad split" }), {
          status: 400,
          headers: { "Content-Type": "application/json" },
        }),
    );
    await enqueue({ op: "createExpense", tripId: "t1", id: "a", body: body("a") });
    await flushQueue(api);
    expect((await failedWrites())[0]?.reason).toBe("Bad split");
    expect(await pendingWrites()).toEqual([]);
  });

  it("does not queue other calls or expenses without an ID", async () => {
    const { api } = client(down);
    const noId: Record<string, unknown> = { ...body("a") };
    delete noId.id;
    await expect(
      api.POST("/v1/trips/{tripId}/expenses", {
        params: { path: { tripId: "t1" } },
        body: noId as never,
      }),
    ).rejects.toThrow();
    expect(await pendingWrites()).toEqual([]);
  });

  it("makes UUID-shaped IDs", () => {
    expect(newId()).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
    );
  });
});
