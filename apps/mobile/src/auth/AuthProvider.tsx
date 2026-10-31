import { localStore } from "@/local/store";
import { useWriteQueue } from "@/local/useWriteQueue";
import { createApiClient, type ApiClient, type StoredTokens } from "@voyagami/api-client";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { reporter } from "@/logging/remoteLog";
import { parseSse } from "@/trips/sse";

import { apiBaseUrl } from "./config";
import { clearDevice, loadOnboarded, loadTokens, saveOnboarded, saveTokens } from "./tokenStore";

/** Leaves a breadcrumb for every API call (method, path without query, status) and nothing else. */
function withBreadcrumbs(base?: typeof fetch): typeof fetch {
  return async (input, init) => {
    const request = input as Request | URL | string;
    const url =
      typeof request === "string" ? request : "url" in request ? request.url : request.href;
    const method = (
      init?.method ?? (typeof request === "object" && "method" in request ? request.method : "GET")
    ).toUpperCase();
    const path = url.replace(/^https?:\/\/[^/]+/, "").replace(/\?.*$/, "");
    try {
      const response = await (base ?? globalThis.fetch)(input, init);
      reporter.breadcrumb(`${method} ${path} ${response.status}`);
      if (response.status >= 500) reporter.log("warn", `${method} ${path} ${response.status}`);
      return response;
    } catch (e) {
      reporter.breadcrumb(`${method} ${path} failed`);
      reporter.log("warn", `${method} ${path} network failure: ${String(e)}`);
      throw e;
    }
  };
}

/**
 * Listens to a trip's change stream and calls onChange when something changed
 * (or when it may have, after a reconnect). Returns a function that stops it.
 * React Native's fetch cannot stream, so this reads the response as it arrives
 * through XMLHttpRequest. It reconnects on its own with a growing pause.
 */
function openTripStream(
  tripId: string,
  token: () => string | null,
  refresh: () => Promise<unknown>,
  onChange: () => void,
): () => void {
  let stopped = false;
  let xhr: XMLHttpRequest | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let pause = 1000;
  let reconnecting = false;

  const connect = () => {
    if (stopped) return;
    const req = new XMLHttpRequest();
    xhr = req;
    let seen = 0;
    let buffer = "";
    req.open("GET", `${apiBaseUrl}/v1/trips/${encodeURIComponent(tripId)}/events`);
    req.setRequestHeader("Accept", "text/event-stream");
    const t = token();
    if (t) req.setRequestHeader("Authorization", `Bearer ${t}`);
    // The server sends a resync first when it sees this header.
    if (reconnecting) req.setRequestHeader("Last-Event-ID", "0");
    req.onprogress = () => {
      if (req.status === 200) pause = 1000;
      buffer += req.responseText.slice(seen);
      seen = req.responseText.length;
      const { events, rest } = parseSse(buffer);
      buffer = rest;
      if (events.some((e) => e.event === "change" || e.event === "resync")) onChange();
    };
    const again = () => {
      if (stopped) return;
      reconnecting = true;
      const wait = pause;
      pause = Math.min(pause * 2, 30000);
      timer = setTimeout(connect, wait);
    };
    req.onerror = again;
    req.onload = () => {
      // 401: the access token is old. A normal call refreshes it; then try again.
      if (req.status === 401) void refresh().finally(again);
      else again();
    };
    req.send();
  };
  connect();
  return () => {
    stopped = true;
    clearTimeout(timer);
    xhr?.abort();
  };
}

export type AuthStatus = "loading" | "signedOut" | "signedIn";

interface AuthContextValue {
  status: AuthStatus;
  api: ApiClient;
  /** Keep the tokens from a successful sign-in. The router then moves to the signed-in screens. */
  signIn: (tokens: StoredTokens) => Promise<void>;
  signOut: () => Promise<void>;
  /** Clears everything stored on this device without calling the server, then shows Welcome. */
  resetDevice: () => Promise<void>;
  /** False until the person finishes or skips through first-run onboarding. */
  onboarded: boolean;
  finishOnboarding: () => Promise<void>;
  /** The number the code was sent to. Held in memory only, so it is not in a route or a log. */
  pendingPhone: string | null;
  setPendingPhone: (phone: string | null) => void;
  /** Calls onChange whenever a trip changes. Returns a stop function. Does nothing in tests. */
  watchTrip: (tripId: string, onChange: () => void) => () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({
  children,
  fetch: fetchImpl,
}: {
  children: ReactNode;
  /** Tests pass a fake. */
  fetch?: typeof fetch;
}) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [pendingPhone, setPendingPhone] = useState<string | null>(null);
  const [onboarded, setOnboarded] = useState(false);
  const tokens = useRef<StoredTokens | null>(null);

  const api = useMemo(
    () =>
      createApiClient({
        baseUrl: apiBaseUrl,
        getTokens: () => tokens.current,
        setTokens: async (next) => {
          tokens.current = next;
          await saveTokens(next);
        },
        onSignedOut: () => setStatus("signedOut"),
        fetch: withBreadcrumbs(fetchImpl),
      }),
    [fetchImpl],
  );
  useWriteQueue(api, status === "signedIn");

  // Auto sign-in on relaunch: saved tokens mean signed in. An expired access token is
  // refreshed on the first request, and a dead session lands back on sign-in via onSignedOut.
  useEffect(() => {
    let alive = true;
    void Promise.all([loadTokens().catch(() => null), loadOnboarded().catch(() => false)]).then(
      ([saved, done]) => {
        if (!alive) return;
        tokens.current = saved;
        setOnboarded(done);
        setStatus(saved ? "signedIn" : "signedOut");
      },
    );
    return () => {
      alive = false;
    };
  }, []);

  const signIn = useCallback(async (next: StoredTokens) => {
    tokens.current = next;
    await saveTokens(next);
    setPendingPhone(null);
    setStatus("signedIn");
  }, []);

  const signOut = useCallback(async () => {
    // Best effort: the device is signed out even if the server cannot be reached.
    await api.POST("/v1/auth/sign-out").catch(() => undefined);
    tokens.current = null;
    await saveTokens(null).catch(() => undefined);
    // Trips cached on the device belong to the person who just left.
    await localStore()
      .clear()
      .catch(() => undefined);
    // The next person to sign in on this device goes through onboarding.
    await saveOnboarded(false).catch(() => undefined);
    setOnboarded(false);
    setStatus("signedOut");
  }, [api]);

  const resetDevice = useCallback(async () => {
    tokens.current = null;
    await clearDevice().catch(() => undefined);
    await localStore()
      .clear()
      .catch(() => undefined);
    setOnboarded(false);
    setStatus("signedOut");
  }, []);

  const finishOnboarding = useCallback(async () => {
    setOnboarded(true);
    await saveOnboarded(true).catch(() => undefined);
  }, []);

  const watchTrip = useCallback(
    (tripId: string, onChange: () => void) => {
      // Tests pass their own fetch; they have no server to stream from.
      if (fetchImpl || typeof XMLHttpRequest === "undefined") return () => undefined;
      return openTripStream(
        tripId,
        () => tokens.current?.accessToken ?? null,
        () => api.GET("/v1/me").catch(() => undefined),
        onChange,
      );
    },
    [api, fetchImpl],
  );

  const value = useMemo(
    () => ({
      status,
      api,
      signIn,
      signOut,
      resetDevice,
      onboarded,
      finishOnboarding,
      pendingPhone,
      setPendingPhone,
      watchTrip,
    }),
    [
      status,
      api,
      signIn,
      signOut,
      resetDevice,
      onboarded,
      finishOnboarding,
      pendingPhone,
      watchTrip,
    ],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
