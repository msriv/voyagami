import Constants from "expo-constants";
import * as SecureStore from "expo-secure-store";
import { Platform } from "react-native";

import { apiBaseUrl } from "@/auth/config";

// shortcut: dev-only crash log shipped to the dev API; remove or point at Sentry in P1.

export type LogLevel = "info" | "warn" | "error";

export interface LogEntry {
  deviceId: string;
  appVersion: string;
  level: LogLevel;
  message: string;
  stack?: string;
  route?: string;
  breadcrumbs?: string[];
}

export interface Reporter {
  breadcrumb: (text: string) => void;
  setRoute: (route: string) => void;
  log: (level: LogLevel, message: string, stack?: string) => void;
}

const maxBreadcrumbs = 30;
const maxText = 2000;

/** Phone numbers and long digit runs never leave the device. */
export function redact(text: string): string {
  return text.replace(/\+?\d[\d\s().-]{6,}\d/g, "[number]").slice(0, maxText);
}

/**
 * Builds a reporter that never throws and never waits: a failed send is dropped, and the
 * last few breadcrumbs ride along with every entry so a crash comes with its trail.
 */
export function createReporter(options: {
  getDeviceId: () => Promise<string>;
  appVersion: string;
  send: (entry: LogEntry) => Promise<unknown>;
}): Reporter {
  const trail: string[] = [];
  let route: string | undefined;

  function breadcrumb(text: string) {
    trail.push(`${new Date().toISOString().slice(11, 19)} ${redact(text)}`);
    if (trail.length > maxBreadcrumbs) trail.shift();
  }

  function log(level: LogLevel, message: string, stack?: string) {
    void (async () => {
      try {
        const entry: LogEntry = {
          deviceId: await options.getDeviceId(),
          appVersion: options.appVersion,
          level,
          message: redact(message),
          breadcrumbs: [...trail],
        };
        if (stack) entry.stack = redact(stack);
        if (route) entry.route = route;
        await options.send(entry);
      } catch {
        // Logging must never break the app.
      }
    })();
  }

  return {
    breadcrumb,
    setRoute: (next) => {
      route = next;
      breadcrumb(`route ${next}`);
    },
    log,
  };
}

const deviceIdKey = "voyagami.deviceId";
let deviceId: string | undefined;

async function getDeviceId(): Promise<string> {
  if (deviceId) return deviceId;
  try {
    deviceId = (await SecureStore.getItemAsync(deviceIdKey)) ?? undefined;
    if (!deviceId) {
      deviceId = `d-${Math.random().toString(36).slice(2, 10)}${Date.now().toString(36)}`;
      await SecureStore.setItemAsync(deviceIdKey, deviceId);
    }
  } catch {
    deviceId ??= `d-${Math.random().toString(36).slice(2, 10)}`;
  }
  return deviceId;
}

const live = process.env.NODE_ENV !== "test";

/** The app-wide reporter. In tests it records breadcrumbs but sends nothing. */
export const reporter: Reporter = createReporter({
  getDeviceId,
  appVersion: Constants.expoConfig?.version ?? "unknown",
  send: (entry) =>
    live
      ? fetch(`${apiBaseUrl}/v1/dev/logs`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(entry),
        })
      : Promise.resolve(),
});

let installed = false;

/** Hooks the global error handler, unhandled rejections and console.error, once per launch. */
export function installRemoteLog(): void {
  if (installed) return;
  installed = true;
  reporter.log("info", `startup ${Platform.OS} ${String(Platform.Version)} api=${apiBaseUrl}`);

  const g = globalThis as unknown as {
    ErrorUtils?: {
      getGlobalHandler: () => (e: unknown, fatal?: boolean) => void;
      setGlobalHandler: (h: (e: unknown, fatal?: boolean) => void) => void;
    };
    HermesInternal?: {
      enablePromiseRejectionTracker?: (o: {
        allRejections: boolean;
        onUnhandled: (id: number, error: unknown) => void;
      }) => void;
    };
  };

  const previous = g.ErrorUtils?.getGlobalHandler();
  g.ErrorUtils?.setGlobalHandler((error, fatal) => {
    reportError(fatal ? "fatal" : "uncaught", error);
    previous?.(error, fatal);
  });

  g.HermesInternal?.enablePromiseRejectionTracker?.({
    allRejections: true,
    onUnhandled: (_id, error) => reportError("unhandled rejection", error),
  });

  const original = console.error;
  console.error = (...args: unknown[]) => {
    reporter.log("error", `console.error ${args.map(String).join(" ")}`);
    original(...args);
  };
}

export function reportError(kind: string, error: unknown): void {
  const e = error instanceof Error ? error : new Error(String(error));
  reporter.log("error", `${kind}: ${e.name}: ${e.message}`, e.stack);
}
