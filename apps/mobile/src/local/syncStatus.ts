import { useEffect, useState } from "react";

import { onQueueChange } from "./changes";
import { failedWrites, pendingWrites } from "./queue";

export type SyncState = "synced" | "syncing" | "offline" | "attention";

export type SyncInfo = { state: SyncState; pending: number; failed: number };

/** Something the server refused beats no signal, which beats waiting to send, which beats all fine. */
export function deriveSync(pending: number, failed: number, offline: boolean): SyncInfo {
  const state: SyncState =
    failed > 0 ? "attention" : offline ? "offline" : pending > 0 ? "syncing" : "synced";
  return { state, pending, failed };
}

// Whether the last call to the server got an answer. Starts as "yes" so the app never claims to be
// offline before it has tried.
let offline = false;
const reachListeners = new Set<() => void>();

export function markReachable(ok: boolean) {
  if (offline === !ok) return;
  offline = !ok;
  reachListeners.forEach((fn) => fn());
}

export function useSyncStatus(): SyncInfo {
  const [counts, setCounts] = useState({ pending: 0, failed: 0 });
  const [off, setOff] = useState(offline);

  useEffect(() => {
    let alive = true;
    const read = () =>
      void Promise.all([pendingWrites(), failedWrites()]).then(([p, f]) => {
        if (alive) setCounts({ pending: p.length, failed: f.length });
      });
    const reach = () => setOff(offline);
    reachListeners.add(reach);
    reach();
    read();
    const stop = onQueueChange(read);
    return () => {
      alive = false;
      reachListeners.delete(reach);
      stop();
    };
  }, []);

  return deriveSync(counts.pending, counts.failed, off);
}
