import { useEffect, useRef } from "react";

import { useAuth } from "@/auth";

/**
 * Runs reload shortly after anything changes on the trip, so a crew member's
 * edit shows up without pulling to refresh. A burst of changes reloads once.
 */
export function useTripChanges(tripId: string | undefined, reload: () => void) {
  const { watchTrip } = useAuth();
  const latest = useRef(reload);
  latest.current = reload;
  useEffect(() => {
    if (!tripId) return;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const stop = watchTrip(tripId, () => {
      clearTimeout(timer);
      timer = setTimeout(() => latest.current(), 300);
    });
    return () => {
      clearTimeout(timer);
      stop();
    };
  }, [tripId, watchTrip]);
}
