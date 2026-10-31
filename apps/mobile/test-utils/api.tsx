import type { ComponentType } from "react";

import RootLayout from "../app/_layout";
import AuthLayout from "../app/(auth)/_layout";
import Code from "../app/(auth)/code";
import Welcome from "../app/(auth)/index";
import Notifications from "../app/notifications";
import ManageSubscription from "../app/manage-subscription";
import Plans from "../app/plans";
import Connect from "../app/onboarding/connect";
import Region from "../app/onboarding/region";
import Style from "../app/onboarding/style";
import SignIn from "../app/(auth)/sign-in";
import TabsLayout from "../app/(tabs)/_layout";
import Explore from "../app/(tabs)/explore";
import Trip from "../app/trips/[id]";
import AddPlace from "../app/trips/[id]/add-place";
import AddExpense from "../app/trips/[id]/add-expense";
import AddBooking from "../app/trips/[id]/add-booking";
import ImportBooking from "../app/trips/[id]/import-booking";
import Bookings from "../app/trips/[id]/bookings";
import Crew from "../app/trips/[id]/crew";
import Comments from "../app/trips/[id]/comments";
import Live from "../app/trips/[id]/live";
import Polls from "../app/trips/[id]/polls";
import Place from "../app/trips/[id]/place";
import Privacy from "../app/privacy";
import Help from "../app/help";
import Ask from "../app/trips/[id]/ask";
import Experiences from "../app/trips/[id]/experiences";
import Picks from "../app/trips/[id]/picks";
import Stays from "../app/trips/[id]/stays";
import Settle from "../app/trips/[id]/settle";
import Sync from "../app/trips/[id]/sync";
import Recap from "../app/trips/[id]/recap";
import RecapPlayer from "../app/trips/[id]/recap-player";
import Join from "../app/join";
import Journal from "../app/trips/[id]/journal";
import Money from "../app/trips/[id]/money";
import Itinerary from "../app/trips/[id]/itinerary";
import MapScreen from "../app/trips/[id]/map";
import NewTrip from "../app/trips/new";
import Home from "../app/(tabs)/index";
import Profile from "../app/(tabs)/profile";

export const routes: Record<string, ComponentType> = {
  _layout: RootLayout,
  "(auth)/_layout": AuthLayout,
  "(auth)/index": Welcome,
  "(auth)/sign-in": SignIn,
  "onboarding/connect": Connect,
  notifications: Notifications,
  plans: Plans,
  "manage-subscription": ManageSubscription,
  "onboarding/region": Region,
  "onboarding/style": Style,
  "(auth)/code": Code,
  "trips/new": NewTrip,
  "trips/[id]": Trip,
  "trips/[id]/itinerary": Itinerary,
  "trips/[id]/map": MapScreen,
  "trips/[id]/add-place": AddPlace,
  "trips/[id]/money": Money,
  "trips/[id]/bookings": Bookings,
  "trips/[id]/journal": Journal,
  "trips/[id]/crew": Crew,
  "trips/[id]/polls": Polls,
  "trips/[id]/stays": Stays,
  "trips/[id]/picks": Picks,
  "trips/[id]/ask": Ask,
  "trips/[id]/experiences": Experiences,
  privacy: Privacy,
  help: Help,
  "trips/[id]/place": Place,
  "trips/[id]/live": Live,
  "trips/[id]/comments": Comments,
  "trips/[id]/settle": Settle,
  "trips/[id]/sync": Sync,
  "trips/[id]/recap": Recap,
  "trips/[id]/recap-player": RecapPlayer,
  join: Join,
  "trips/[id]/add-booking": AddBooking,
  "trips/[id]/import-booking": ImportBooking,
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
