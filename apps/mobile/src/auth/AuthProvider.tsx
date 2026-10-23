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
import { loadTokens, saveTokens } from "./tokenStore";

export type AuthStatus = "loading" | "signedOut" | "signedIn";

interface AuthContextValue {
  status: AuthStatus;
  api: ApiClient;
  /** Keep the tokens from a successful sign-in. The router then moves to the signed-in screens. */
  signIn: (tokens: StoredTokens) => Promise<void>;
  signOut: () => Promise<void>;
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
    void loadTokens()
      .catch(() => null)
      .then((saved) => {
        if (!alive) return;
        tokens.current = saved;
        setStatus(saved ? "signedIn" : "signedOut");
      });
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
    setStatus("signedOut");
  }, [api]);

  const value = useMemo(
    () => ({ status, api, signIn, signOut, pendingPhone, setPendingPhone }),
    [status, api, signIn, signOut, pendingPhone],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
