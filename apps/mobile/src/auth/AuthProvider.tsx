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

import { apiBaseUrl } from "./config";
import { loadOnboarded, loadTokens, saveOnboarded, saveTokens } from "./tokenStore";

export type AuthStatus = "loading" | "signedOut" | "signedIn";

interface AuthContextValue {
  status: AuthStatus;
  api: ApiClient;
  /** Keep the tokens from a successful sign-in. The router then moves to the signed-in screens. */
  signIn: (tokens: StoredTokens) => Promise<void>;
  signOut: () => Promise<void>;
  /** False until the person finishes or skips through first-run onboarding. */
  onboarded: boolean;
  finishOnboarding: () => Promise<void>;
  /** The number the code was sent to. Held in memory only, so it is not in a route or a log. */
  pendingPhone: string | null;
  setPendingPhone: (phone: string | null) => void;
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
        ...(fetchImpl ? { fetch: fetchImpl } : null),
      }),
    [fetchImpl],
  );

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
    // The next person to sign in on this device goes through onboarding.
    await saveOnboarded(false).catch(() => undefined);
    setOnboarded(false);
    setStatus("signedOut");
  }, [api]);

  const finishOnboarding = useCallback(async () => {
    setOnboarded(true);
    await saveOnboarded(true).catch(() => undefined);
  }, []);

  const value = useMemo(
    () => ({
      status,
      api,
      signIn,
      signOut,
      onboarded,
      finishOnboarding,
      pendingPhone,
      setPendingPhone,
    }),
    [status, api, signIn, signOut, onboarded, finishOnboarding, pendingPhone],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
