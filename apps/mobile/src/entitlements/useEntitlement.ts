import { useEffect, useState } from "react";

import { useAuth } from "@/auth";

export interface Entitlement {
  /** False until the server has answered; the app only reflects the server's answer. */
  loaded: boolean;
  allowed: boolean;
  /** The cap, -1 for unlimited, 0 when a flag feature is closed. */
  limit: number;
  used: number;
  /** True when Plus would open it. */
  needsPlus: boolean;
}

const unknown: Entitlement = { loaded: false, allowed: false, limit: 0, used: 0, needsPlus: false };

/**
 * What the server says about one feature. Pass a trip id for trip features, so a Trip Pass or a
 * crew member's Plus counts. Until the answer arrives (or if it fails) nothing is allowed.
 */
export function useEntitlement(feature: string, tripId?: string): Entitlement {
  const { api, status } = useAuth();
  const [answer, setAnswer] = useState<Entitlement>(unknown);

  useEffect(() => {
    if (status !== "signedIn") return;
    let live = true;
    const load = tripId
      ? api.unwrap(api.GET("/v1/trips/{tripId}/entitlements", { params: { path: { tripId } } }))
      : api.unwrap(api.GET("/v1/me/entitlements"));
    load
      .then((e) => {
        const d = e.features[feature];
        if (live) setAnswer(d ? { loaded: true, ...d } : unknown);
      })
      .catch(() => {
        if (live) setAnswer(unknown);
      });
    return () => {
      live = false;
    };
  }, [api, status, feature, tripId]);

  return answer;
}
