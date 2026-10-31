import { useEffect, useState } from "react";

import { useAuth } from "@/auth";

/**
 * True when anything on the trip is over its Free limit (after a downgrade). That content is kept
 * and can be read, but not changed. The server decides; this only reflects its answer.
 */
export function useTripReadOnly(tripId: string): boolean {
  const { api } = useAuth();
  const [readOnly, setReadOnly] = useState(false);

  useEffect(() => {
    api
      .unwrap(api.GET("/v1/trips/{tripId}/entitlements", { params: { path: { tripId } } }))
      .then((e) => setReadOnly(Object.values(e.features).some((f) => f.readOnly)))
      .catch(() => setReadOnly(false));
  }, [api, tripId]);

  return readOnly;
}
