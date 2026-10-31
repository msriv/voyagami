import createClient from "openapi-fetch";

import { ApiError, NetworkError, type ApiErrorBody } from "./errors";
import type { components, paths } from "./generated/schema";

export type SessionTokens = components["schemas"]["SessionTokens"];

/** What the app keeps between launches. Store the refresh token in secure storage. */
export interface StoredTokens {
  accessToken: string;
  refreshToken: string;
}

/**
 * Hook point for the Offline & Sync project. When a request cannot reach the
 * server, the client calls this before failing. Return a Response to stand in
 * for the missing answer (for example after queueing the write), or
 * `undefined` to let the failure surface as a NetworkError.
 */
export interface OfflineHook {
  /** Cheap, synchronous check; when it says no, the failure surfaces without waiting on the hook. */
  handles?(request: Request): boolean;
  onNetworkError(request: Request, error: unknown): Promise<Response | undefined>;
}

export interface ApiClientOptions {
  baseUrl: string;
  getTokens: () => StoredTokens | null | Promise<StoredTokens | null>;
  /** Called with new tokens after a refresh, and with `null` when the session has ended. */
  setTokens: (tokens: StoredTokens | null) => void | Promise<void>;
  /** Called once when a refresh fails and the person must sign in again. */
  onSignedOut?: () => void;
  offline?: OfflineHook;
  /** Defaults to the global fetch. Tests pass a fake. */
  fetch?: typeof fetch;
}

/** Routes that need no token and must never trigger a refresh. */
const PUBLIC_PATHS = new Set(["/v1/auth/otp/request", "/v1/auth/otp/verify", "/v1/auth/refresh"]);

/**
 * Creates the API client. Everything about talking to the server lives here:
 * the Authorization header, one-at-a-time token refresh with a single retry,
 * and the offline hook. Use `unwrap` to turn error answers into ApiError.
 */
export function createApiClient(options: ApiClientOptions) {
  const baseFetch: typeof fetch = options.fetch ?? ((...args) => fetch(...args));
  let refreshing: Promise<StoredTokens | null> | null = null;

  async function send(request: Request): Promise<Response> {
    const retryCopy = request.clone();
    try {
      return await baseFetch(request);
    } catch (error) {
      const standIn =
        options.offline && (options.offline.handles?.(retryCopy) ?? true)
          ? await options.offline.onNetworkError(retryCopy, error)
          : undefined;
      if (standIn) return standIn;
      throw new NetworkError(error);
    }
  }

  async function refresh(): Promise<StoredTokens | null> {
    const current = await options.getTokens();
    if (!current) return null;
    const response = await send(
      new Request(new URL("/v1/auth/refresh", options.baseUrl), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken: current.refreshToken }),
      }),
    );
    if (!response.ok) {
      await options.setTokens(null);
      options.onSignedOut?.();
      return null;
    }
    const body = (await response.json()) as SessionTokens;
    const next = { accessToken: body.accessToken, refreshToken: body.refreshToken };
    await options.setTokens(next);
    return next;
  }

  /** One refresh at a time: a burst of 401s shares one refresh, because a refresh token works once. */
  function refreshOnce(): Promise<StoredTokens | null> {
    refreshing ??= refresh().finally(() => {
      refreshing = null;
    });
    return refreshing;
  }

  const authedFetch = async (input: Request): Promise<Response> => {
    const isPublic = PUBLIC_PATHS.has(new URL(input.url).pathname);
    if (isPublic) return send(input);

    const used = await options.getTokens();
    if (used) input.headers.set("Authorization", `Bearer ${used.accessToken}`);
    const retryCopy = input.clone();
    const response = await send(input);
    if (response.status !== 401 || !used) return response;

    // Another request may have refreshed already; then just retry with its tokens.
    const latest = await options.getTokens();
    const next = latest && latest.accessToken !== used.accessToken ? latest : await refreshOnce();
    if (!next) return response;
    retryCopy.headers.set("Authorization", `Bearer ${next.accessToken}`);
    return send(retryCopy);
  };

  const client = createClient<paths>({ baseUrl: options.baseUrl, fetch: authedFetch });

  /**
   * Await a client call and return its data, or throw ApiError for an error
   * answer. Example: `const me = await unwrap(api.GET("/v1/me"))`.
   */
  async function unwrap<T>(
    call: Promise<{ data?: T; error?: unknown; response: Response }>,
  ): Promise<T> {
    const { data, error, response } = await call;
    if (!response.ok) {
      const body =
        typeof error === "object" && error !== null ? (error as Partial<ApiErrorBody>) : undefined;
      throw new ApiError(response.status, body);
    }
    return data as T;
  }

  return { ...client, unwrap };
}

export type ApiClient = ReturnType<typeof createApiClient>;
