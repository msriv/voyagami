import { AppState } from "react-native";
import { act, renderRouter, waitFor } from "expo-router/testing-library";

import { json, mockApi, routes } from "../test-utils/api";

const mockStore = new Map<string, string>();
jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn((k: string) => Promise.resolve(mockStore.get(k) ?? null)),
  setItemAsync: jest.fn((k: string, v: string) => Promise.resolve(void mockStore.set(k, v))),
  deleteItemAsync: jest.fn((k: string) => Promise.resolve(void mockStore.delete(k))),
}));
jest.mock("expo-font", () => ({ useFonts: () => [true, null] }));
jest.mock("expo-splash-screen", () => ({
  preventAutoHideAsync: jest.fn(() => Promise.resolve()),
  hideAsync: jest.fn(() => Promise.resolve()),
}));

const handlers: ((s: string) => void)[] = [];
beforeEach(() => {
  mockStore.clear();
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
  mockStore.set("voyagami.onboarded", "1");
  mockStore.set("voyagami.tour", "done");
  handlers.length = 0;
  jest.spyOn(AppState, "addEventListener").mockImplementation((_t, fn) => {
    handlers.push(fn as (s: string) => void);
    return { remove: () => undefined };
  });
});

async function start(consent: boolean) {
  if (consent) mockStore.set("voyagami.consents", JSON.stringify({ analytics: true }));
  const calls = mockApi({
    "GET /v1/me": () => json(200, { id: "u1" }),
    "GET /v1/trips": () => json(200, []),
    "POST /v1/events": () => json(200, { accepted: 1, dropped: 0 }),
  });
  await renderRouter(routes, { initialUrl: "/" });
  await waitFor(() => expect(calls.some((c) => c.key === "GET /v1/me")).toBe(true));
  return calls;
}

describe("Analytics", () => {
  it("sends an app_open event when the app goes to the background, with no personal data", async () => {
    const calls = await start(true);
    await waitFor(() => expect(handlers.length).toBeGreaterThan(0));
    await act(async () => handlers.forEach((h) => h("background")));
    await waitFor(() => expect(calls.some((c) => c.key === "POST /v1/events")).toBe(true));
    const body = calls.find((c) => c.key === "POST /v1/events")?.body as {
      events: { name: string }[];
    };
    expect(body.events.map((e) => e.name)).toContain("app_open");
    expect(JSON.stringify(body)).not.toMatch(/phone|lat|lng/);
  });

  it("sends nothing when analytics is off", async () => {
    const calls = await start(false);
    await act(async () => handlers.forEach((h) => h("background")));
    expect(calls.some((c) => c.key === "POST /v1/events")).toBe(false);
  });
});
