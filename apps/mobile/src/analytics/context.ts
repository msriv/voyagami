import Constants from "expo-constants";
import { Platform } from "react-native";

import { i18n } from "@/i18n";

const sessionGapMs = 30 * 60 * 1000;

function randomId(): string {
  let s = "";
  for (let i = 0; i < 32; i++) s += Math.floor(Math.random() * 16).toString(16);
  return s;
}

let sessionId = randomId();
let backgroundAt: number | null = null;

/** Starts a new session after 30 minutes in the background. */
export function noteAppState(active: boolean, now = Date.now()): void {
  if (!active) {
    backgroundAt = now;
    return;
  }
  if (backgroundAt !== null && now - backgroundAt > sessionGapMs) sessionId = randomId();
  backgroundAt = null;
}

/** What every batch says about the app. Identifiers only; the server drops anything else. */
export function eventContext() {
  return {
    sessionId,
    platform: Platform.OS,
    osVersion: String(Platform.Version),
    appVersion: Constants.expoConfig?.version ?? "",
    appLanguage: i18n.language,
  };
}

/** Expo Router path to an analytics route: ids dropped, dots for slashes ("/trips/abc/money" is "trips.id.money"). */
export function routeName(path: string, params: Record<string, unknown> = {}): string {
  const ids = new Set(Object.values(params).filter((v): v is string => typeof v === "string"));
  const parts = path
    .split("/")
    .filter(Boolean)
    .map((p) => (ids.has(p) ? "id" : p));
  return parts.length ? parts.join(".") : "home";
}
