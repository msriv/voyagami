import type { ApiClient } from "@voyagami/api-client";
import { useEffect } from "react";
import { AppState } from "react-native";

import { flushQueue } from "./offlineHook";

const EVERY_MS = 15_000;

/** Sends queued writes now, when the app comes back to the front, and every 15 seconds. */
export function useWriteQueue(api: ApiClient, active: boolean) {
  useEffect(() => {
    if (!active) return;
    let busy = false;
    const run = () => {
      if (busy) return;
      busy = true;
      flushQueue(api)
        .catch(() => undefined)
        .finally(() => (busy = false));
    };
    run();
    const timer = setInterval(run, EVERY_MS);
    const sub = AppState.addEventListener("change", (s) => s === "active" && run());
    return () => {
      clearInterval(timer);
      sub.remove();
    };
  }, [api, active]);
}
