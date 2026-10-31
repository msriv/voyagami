import { ApiError, NetworkError } from "@voyagami/api-client";

import { localFirst } from "./localFirst";
import { localStore, memoryStore, setLocalStore } from "./store";

beforeEach(() => setLocalStore(memoryStore()));

const seen = () => {
  const calls: [string, boolean][] = [];
  return { calls, onData: (d: string, cached: boolean) => calls.push([d, cached]) };
};

describe("localFirst", () => {
  it("shows the device copy, then the fresh answer, and keeps the fresh one", async () => {
    await localStore().put("k", "old");
    const { calls, onData } = seen();
    await localFirst("k", async () => "new", onData);
    expect(calls).toEqual([
      ["old", true],
      ["new", false],
    ]);
    await new Promise((r) => setTimeout(r, 0));
    expect(await localStore().get("k")).toBe("new");
  });

  it("keeps the device copy when the network or server fails", async () => {
    await localStore().put("k", "old");
    const { calls, onData } = seen();
    await localFirst("k", () => Promise.reject(new NetworkError(new Error("offline"))), onData);
    await localFirst("k", () => Promise.reject(new ApiError(500, { code: "internal" })), onData);
    expect(calls).toEqual([
      ["old", true],
      ["old", true],
    ]);
    expect(await localStore().get("k")).toBe("old");
  });

  it("throws when there is no copy to fall back on", async () => {
    const { onData } = seen();
    await expect(
      localFirst("k", () => Promise.reject(new NetworkError(new Error("offline"))), onData),
    ).rejects.toBeInstanceOf(NetworkError);
  });

  it("drops the copy and throws when the trip is gone", async () => {
    await localStore().put("k", "old");
    const { onData } = seen();
    await expect(
      localFirst("k", () => Promise.reject(new ApiError(404, { code: "not_found" })), onData),
    ).rejects.toBeInstanceOf(ApiError);
    await new Promise((r) => setTimeout(r, 0));
    expect(await localStore().get("k")).toBeUndefined();
  });

  it("works when the local store itself is broken", async () => {
    setLocalStore({ ...memoryStore(), get: () => Promise.reject(new Error("disk")) });
    const { calls, onData } = seen();
    await localFirst("k", async () => "new", onData);
    expect(calls).toEqual([["new", false]]);
  });
});
