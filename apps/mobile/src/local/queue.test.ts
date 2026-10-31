import { backoffMs, enqueue, failedWrites, flush, pendingWrites, type Outcome } from "./queue";
import { memoryStore, setLocalStore } from "./store";

beforeEach(() => setLocalStore(memoryStore()));

const body = {
  description: "Pho",
  paidByUid: "u1",
  date: "2099-11-19",
  original: { amountMinor: 100, currency: "VND" },
  split: { method: "equal" as const, parts: [{ uid: "u1" }] },
};
const add = (id: string, at = 0) =>
  enqueue({ op: "createExpense", tripId: "t1", id, body: { ...body, id } }, at);

describe("write queue", () => {
  it("keeps one entry per ID", async () => {
    await add("a");
    await add("a");
    expect(await pendingWrites()).toHaveLength(1);
  });

  it("sends in order and empties the queue", async () => {
    await add("a");
    await add("b");
    const order: string[] = [];
    const n = await flush(async (e) => (order.push(e.id), "sent"), 0);
    expect(n).toBe(2);
    expect(order).toEqual(["a", "b"]);
    expect(await pendingWrites()).toEqual([]);
  });

  it("stops at a failure so later writes never pass it, and backs off", async () => {
    await add("a");
    await add("b");
    const seen: string[] = [];
    const flaky = async (e: { id: string }): Promise<Outcome> => (seen.push(e.id), "retry");
    expect(await flush(flaky, 0)).toBe(0);
    expect(seen).toEqual(["a"]);
    const [a, b] = await pendingWrites();
    expect([a?.attempts, a?.nextAt, b?.attempts]).toEqual([1, 2000, 0]);
    seen.length = 0;
    await flush(flaky, 1000);
    expect(seen).toEqual([]);
    await flush(flaky, 2000);
    expect(seen).toEqual(["a"]);
    expect((await pendingWrites())[0]?.nextAt).toBe(2000 + 4000);
  });

  it("treats a thrown error as a retry", async () => {
    await add("a");
    await flush(() => Promise.reject(new Error("socket")), 0);
    expect((await pendingWrites())[0]?.attempts).toBe(1);
  });

  it("moves a rejected write to the failed list and carries on", async () => {
    await add("a");
    await add("b");
    const n = await flush(async (e) => (e.id === "a" ? { reject: "Not allowed" } : "sent"), 0);
    expect(n).toBe(1);
    expect(await pendingWrites()).toEqual([]);
    expect((await failedWrites()).map((f) => [f.id, f.reason])).toEqual([["a", "Not allowed"]]);
  });

  it("caps the wait at five minutes", () => {
    expect(backoffMs(1)).toBe(2000);
    expect(backoffMs(30)).toBe(300_000);
  });
});
