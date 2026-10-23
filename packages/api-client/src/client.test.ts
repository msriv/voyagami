import { describe, expect, it, vi } from "vitest";

import { ApiError, NetworkError, createApiClient, type StoredTokens } from "./index";

const BASE = "https://api.test";

const json = (status: number, body: unknown): Response =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

const tokensBody = (n: number) => ({
  accessToken: `a${n}`,
  refreshToken: `r${n}`,
  expiresIn: 900,
  userId: "u1",
  isNewUser: false,
});

function setup(
  handler: (req: Request) => Response | Promise<Response>,
  start: StoredTokens | null,
) {
  let tokens = start;
  const calls: { path: string; auth: string | null }[] = [];
  const fakeFetch = vi.fn(async (input: RequestInfo | URL) => {
    const req = input as Request;
    calls.push({ path: new URL(req.url).pathname, auth: req.headers.get("Authorization") });
    return handler(req);
  });
  const setTokens = vi.fn((t: StoredTokens | null) => {
    tokens = t;
  });
  const onSignedOut = vi.fn();
  const api = createApiClient({
    baseUrl: BASE,
    getTokens: () => tokens,
    setTokens,
    onSignedOut,
    fetch: fakeFetch as unknown as typeof fetch,
  });
  return { api, calls, setTokens, onSignedOut, getTokens: () => tokens };
}

describe("auth header", () => {
  it("sends the access token on protected routes", async () => {
    const t = setup(() => json(200, { id: "u1" }), { accessToken: "a0", refreshToken: "r0" });
    const me = await t.api.unwrap(t.api.GET("/v1/me"));
    expect(me).toEqual({ id: "u1" });
    expect(t.calls).toEqual([{ path: "/v1/me", auth: "Bearer a0" }]);
  });

  it("sends no token on sign-in routes", async () => {
    const t = setup(() => new Response(null, { status: 204 }), {
      accessToken: "a0",
      refreshToken: "r0",
    });
    await t.api.POST("/v1/auth/otp/request", { body: { phone: "+14155550123" } });
    expect(t.calls[0]?.auth).toBeNull();
  });
});

describe("token refresh", () => {
  it("refreshes once on 401, stores the new tokens and retries", async () => {
    const t = setup(
      (req) => {
        const path = new URL(req.url).pathname;
        if (path === "/v1/auth/refresh") return json(200, tokensBody(1));
        return req.headers.get("Authorization") === "Bearer a1"
          ? json(200, { id: "u1" })
          : json(401, { code: "unauthenticated", message: "no" });
      },
      { accessToken: "a0", refreshToken: "r0" },
    );
    const me = await t.api.unwrap(t.api.GET("/v1/me"));
    expect(me).toEqual({ id: "u1" });
    expect(t.setTokens).toHaveBeenCalledWith({ accessToken: "a1", refreshToken: "r1" });
    expect(t.calls.map((c) => c.path)).toEqual(["/v1/me", "/v1/auth/refresh", "/v1/me"]);
  });

  it("shares one refresh between simultaneous 401s", async () => {
    let refreshes = 0;
    const t = setup(
      async (req) => {
        const path = new URL(req.url).pathname;
        if (path === "/v1/auth/refresh") {
          refreshes++;
          await new Promise((r) => setTimeout(r, 10));
          return json(200, tokensBody(1));
        }
        return req.headers.get("Authorization") === "Bearer a1"
          ? json(200, { id: "u1" })
          : json(401, { code: "unauthenticated", message: "no" });
      },
      { accessToken: "a0", refreshToken: "r0" },
    );
    const results = await Promise.all([
      t.api.unwrap(t.api.GET("/v1/me")),
      t.api.unwrap(t.api.GET("/v1/me")),
      t.api.unwrap(t.api.GET("/v1/me")),
    ]);
    expect(results).toHaveLength(3);
    expect(refreshes).toBe(1);
  });

  it("clears tokens and signals sign-out when the refresh fails", async () => {
    const t = setup(
      (req) =>
        new URL(req.url).pathname === "/v1/auth/refresh"
          ? json(401, { code: "unauthenticated", message: "used" })
          : json(401, { code: "unauthenticated", message: "expired" }),
      { accessToken: "a0", refreshToken: "r0" },
    );
    await expect(t.api.unwrap(t.api.GET("/v1/me"))).rejects.toMatchObject({
      status: 401,
      code: "unauthenticated",
    });
    expect(t.getTokens()).toBeNull();
    expect(t.onSignedOut).toHaveBeenCalledTimes(1);
  });

  it("does not try to refresh when there are no tokens", async () => {
    const t = setup(() => json(401, { code: "unauthenticated", message: "no" }), null);
    await expect(t.api.unwrap(t.api.GET("/v1/me"))).rejects.toBeInstanceOf(ApiError);
    expect(t.calls.map((c) => c.path)).toEqual(["/v1/me"]);
  });

  it("does not retry a second time if the retry is also rejected", async () => {
    const t = setup(
      (req) =>
        new URL(req.url).pathname === "/v1/auth/refresh"
          ? json(200, tokensBody(1))
          : json(401, { code: "unauthenticated", message: "no" }),
      { accessToken: "a0", refreshToken: "r0" },
    );
    await expect(t.api.unwrap(t.api.GET("/v1/me"))).rejects.toMatchObject({ status: 401 });
    expect(t.calls.filter((c) => c.path === "/v1/me")).toHaveLength(2);
  });
});

describe("error mapping", () => {
  it("turns an error answer into ApiError with code, request id and field errors", async () => {
    const t = setup(
      () =>
        json(400, {
          code: "bad_request",
          message: "Bad",
          requestId: "req1",
          fieldErrors: [{ field: "phone", message: "x" }],
        }),
      null,
    );
    const err = await t.api
      .unwrap(t.api.POST("/v1/auth/otp/request", { body: { phone: "x" } }))
      .catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({
      status: 400,
      code: "bad_request",
      requestId: "req1",
      message: "Bad",
    });
    expect((err as ApiError).fieldErrors).toEqual([{ field: "phone", message: "x" }]);
  });

  it("survives a non-JSON error body", async () => {
    const t = setup(() => new Response("gateway timeout", { status: 504 }), null);
    await expect(t.api.unwrap(t.api.GET("/health"))).rejects.toMatchObject({
      status: 504,
      code: "unknown",
    });
  });
});

describe("offline hook", () => {
  const failing = () => Promise.reject(new TypeError("network down"));

  it("throws NetworkError when there is no hook", async () => {
    const api = createApiClient({
      baseUrl: BASE,
      getTokens: () => null,
      setTokens: () => undefined,
      fetch: failing as unknown as typeof fetch,
    });
    await expect(api.GET("/health")).rejects.toBeInstanceOf(NetworkError);
  });

  it("lets the hook answer for the missing server", async () => {
    const seen: string[] = [];
    const api = createApiClient({
      baseUrl: BASE,
      getTokens: () => null,
      setTokens: () => undefined,
      fetch: failing as unknown as typeof fetch,
      offline: {
        onNetworkError: async (req) => {
          seen.push(`${req.method} ${new URL(req.url).pathname}`);
          return new Response(null, { status: 204 });
        },
      },
    });
    const { response } = await api.POST("/v1/auth/otp/request", {
      body: { phone: "+14155550123" },
    });
    expect(response.status).toBe(204);
    expect(seen).toEqual(["POST /v1/auth/otp/request"]);
  });
});
