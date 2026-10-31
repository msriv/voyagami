import { useEffect, useState } from "react";

import { useAuth } from "@/auth";

export interface TripPlan {
  /** What covers the trip for the caller. `pass` only when the trip itself has a Trip Pass. */
  plan: "free" | "plus" | "pass" | undefined;
  /** True when anything on the trip is over its Free limit (after a downgrade). Kept, readable, not changeable. */
  readOnly: boolean;
  /** Who bought the pass and its last day, when the trip has one. */
  pass: { boughtBy: string; endsOn?: string } | undefined;
}

/** The server decides; this only reflects its answer for the trip. */
export function useTripPlan(tripId: string): TripPlan {
  const { api } = useAuth();
  const [state, setState] = useState<TripPlan>({
    plan: undefined,
    readOnly: false,
    pass: undefined,
  });

  useEffect(() => {
    api
      .unwrap(api.GET("/v1/trips/{tripId}/entitlements", { params: { path: { tripId } } }))
      .then((e) =>
        setState({
          plan: e.plan,
          readOnly: Object.values(e.features).some((f) => f.readOnly),
          pass: e.pass,
        }),
      )
      .catch(() => setState({ plan: undefined, readOnly: false, pass: undefined }));
  }, [api, tripId]);

  return state;
}

/** The Trip Pass that fits a trip of this many days, matching the lengths on the Plans screen. */
export function passFor(days: number): "weekend" | "week" | "long" {
  return days <= 4 ? "weekend" : days <= 10 ? "week" : "long";
}
