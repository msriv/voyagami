import type { ComponentType } from "react";

import RootLayout from "../app/_layout";
import AuthLayout from "../app/(auth)/_layout";
import Code from "../app/(auth)/code";
import SignIn from "../app/(auth)/sign-in";
import TabsLayout from "../app/(tabs)/_layout";
import Explore from "../app/(tabs)/explore";
import Home from "../app/(tabs)/index";
import Profile from "../app/(tabs)/profile";

export const routes: Record<string, ComponentType> = {
  _layout: RootLayout,
  "(auth)/_layout": AuthLayout,
  "(auth)/sign-in": SignIn,
  "(auth)/code": Code,
  "(tabs)/_layout": TabsLayout,
  "(tabs)/index": Home,
  "(tabs)/explore": Explore,
  "(tabs)/profile": Profile,
};

export const json = (status: number, body?: unknown) =>
  new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

type Handler = (request: Request) => Response | Promise<Response>;

/** Replaces global fetch with a router keyed by "METHOD /path". Unknown routes fail the test. */
export function mockApi(handlers: Record<string, Handler>) {
  const calls: { key: string; body: unknown }[] = [];
  globalThis.fetch = jest.fn(async (input: RequestInfo | URL) => {
    const request = input as Request;
    const key = `${request.method} ${new URL(request.url).pathname}`;
    const handler = handlers[key];
    if (!handler) throw new Error(`Unexpected request: ${key}`);
    const text = await request.clone().text();
    calls.push({ key, body: text ? JSON.parse(text) : undefined });
    return handler(request);
  }) as unknown as typeof fetch;
  return calls;
}

export const tokens = {
  accessToken: "access-1",
  refreshToken: "refresh-1",
  expiresIn: 900,
  userId: "u1",
  isNewUser: true,
};
