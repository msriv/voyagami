import { useEffect } from "react";
import { AppState } from "react-native";

import { useAuth } from "@/auth";
import { useConsent } from "@/privacy/Consent";

import { onTracked, putBack, setTrackingEnabled, takeEvents, track } from "./track";

const BATCH = 20;

/** Sends waiting events in batches while signed in and while analytics consent is on. */
export function AnalyticsSender() {
  const { api, status } = useAuth();
  const { choices } = useConsent();
  const on = choices.analytics && status === "signedIn";

  useEffect(() => {
    setTrackingEnabled(on);
    if (!on) return;
    track("app_open");
    let busy = false;
    const flush = async (all: boolean) => {
      if (busy) return;
      busy = true;
      try {
        for (;;) {
          const events = takeEvents(BATCH);
          if (events.length === 0 || (!all && events.length < BATCH)) {
            if (events.length) putBack(events);
            return;
          }
          const res = await api.POST("/v1/events", { body: { events } }).catch(() => undefined);
          if (!res || res.error) {
            putBack(events);
            return;
          }
        }
      } finally {
        busy = false;
      }
    };
    const stop = onTracked(() => void flush(false));
    const sub = AppState.addEventListener("change", (s) => {
      if (s !== "active") void flush(true);
    });
    return () => {
      stop();
      sub.remove();
      setTrackingEnabled(false);
    };
  }, [api, on]);

  return null;
}
