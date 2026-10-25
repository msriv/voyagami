import type { ComponentType } from "react";

import RootLayout from "../app/_layout";
import AuthLayout from "../app/(auth)/_layout";
import Code from "../app/(auth)/code";
import Welcome from "../app/(auth)/index";
import Connect from "../app/onboarding/connect";
import Region from "../app/onboarding/region";
import Style from "../app/onboarding/style";
import SignIn from "../app/(auth)/sign-in";
import TabsLayout from "../app/(tabs)/_layout";
import Explore from "../app/(tabs)/explore";
import Trip from "../app/trips/[id]";
import AddPlace from "../app/trips/[id]/add-place";
import AddExpense from "../app/trips/[id]/add-expense";
import Money from "../app/trips/[id]/money";
import Itinerary from "../app/trips/[id]/itinerary";
import NewTrip from "../app/trips/new";
import Home from "../app/(tabs)/index";
import Profile from "../app/(tabs)/profile";

export const routes: Record<string, ComponentType> = {
  _layout: RootLayout,
  "(auth)/_layout": AuthLayout,
  "(auth)/index": Welcome,
  "(auth)/sign-in": SignIn,
  "onboarding/connect": Connect,
  "onboarding/region": Region,
  "onboarding/style": Style,
  "(auth)/code": Code,
  "trips/new": NewTrip,
  "trips/[id]": Trip,
  "trips/[id]/itinerary": Itinerary,
  "trips/[id]/add-place": AddPlace,
  "trips/[id]/money": Money,
  "trips/[id]/add-expense": AddExpense,
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
