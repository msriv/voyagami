import { useGlobalSearchParams, usePathname } from "expo-router";
import * as SecureStore from "expo-secure-store";
import { useEffect, useRef } from "react";
import { AppState } from "react-native";

import { useAuth } from "@/auth";
import { useConsent } from "@/privacy/Consent";

import { eventContext, noteAppState, routeName } from "./context";
import { onTracked, putBack, setTrackingEnabled, takeEvents, track } from "./track";

const BATCH = 20;
const installedKey = "voyagami.analyticsInstalled";
let coldStart = true;

/** Sends waiting events in batches while signed in and while analytics consent is on. */
export function AnalyticsSender() {
  const { api, status } = useAuth();
  const { choices } = useConsent();
  const on = choices.analytics && status === "signedIn";

  useEffect(() => {
    setTrackingEnabled(on);
    if (!on) return;
    track("app_open", { coldStart });
    coldStart = false;
    void SecureStore.getItemAsync(installedKey).then((v) => {
      if (v) return;
      track("app_installed");
      return SecureStore.setItemAsync(installedKey, "1");
    });
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
          const res = await api
            .POST("/v1/events", { body: { events, context: eventContext() } })
            .catch(() => undefined);
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
      noteAppState(s === "active");
      if (s !== "active") void flush(true);
    });
    return () => {
      stop();
      sub.remove();
      setTrackingEnabled(false);
    };
  }, [api, on]);

  return <ScreenViews />;
}

/** screen_viewed on every route change (decided 2026-11-02: every route). */
function ScreenViews() {
  const path = usePathname();
  const params = useGlobalSearchParams();
  const previous = useRef<string | undefined>(undefined);
  useEffect(() => {
    const route = routeName(path, params);
    if (route === previous.current) return;
    track(
      "screen_viewed",
      previous.current ? { route, previousRoute: previous.current } : { route },
    );
    previous.current = route;
    // params change with the path; reading them is enough
  }, [path]);
  return null;
}
