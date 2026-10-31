import { ApiError } from "@voyagami/api-client";

import { reportError } from "@/logging/remoteLog";

import { localStore } from "./store";

/**
 * Local-first read. Shows what the device already has at once (`onData(cached, true)`), then asks
 * the server and shows and keeps the fresh answer (`onData(fresh, false)`).
 *
 * If the fresh read fails (offline, server error) and the device had a copy, the copy stays and
 * nothing is thrown. "Not found" and "forbidden" are thrown and drop the copy, because the trip is
 * gone or no longer shared. Local store trouble never blocks the screen.
 */
export async function localFirst<T>(
  key: string,
  fetchFresh: () => Promise<T>,
  onData: (data: T, fromCache: boolean) => void,
): Promise<void> {
  let had = false;
  try {
    const cached = (await localStore().get(key)) as T | undefined;
    if (cached !== undefined) {
      had = true;
      onData(cached, true);
    }
  } catch (e) {
    reportError("local-read", e);
  }
  try {
    const fresh = await fetchFresh();
    onData(fresh, false);
    localStore()
      .put(key, fresh)
      .catch((e) => reportError("local-write", e));
  } catch (e) {
    const gone = e instanceof ApiError && (e.code === "not_found" || e.code === "forbidden");
    if (gone)
      void localStore()
        .remove(key)
        .catch(() => undefined);
    if (had && !gone) return;
    throw e;
  }
}
