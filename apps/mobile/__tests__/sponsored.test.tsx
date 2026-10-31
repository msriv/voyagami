import { Linking } from "react-native";
import { fireEvent, renderRouter, screen, waitFor } from "expo-router/testing-library";

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

beforeEach(() => {
  mockStore.clear();
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
  mockStore.set("voyagami.onboarded", "1");
  mockStore.set("voyagami.tour", "done");
});

const card = {
  id: "c1",
  sponsor: "Cafe Co",
  title: "Free coffee",
  url: "https://cafe.example.com",
};

function api(result: unknown) {
  return mockApi({
    "GET /v1/me": () => json(200, { id: "u1" }),
    "GET /v1/trips/t1/stays/links": () => json(200, { links: [] }),
    "GET /v1/trips/t1/sponsored": () => json(200, result),
    "POST /v1/sponsored/c1/impressions": () => new Response(null, { status: 204 }),
    "POST /v1/sponsored/c1/clicks": () => new Response(null, { status: 204 }),
    "POST /v1/sponsored/c1/hide": () => json(200, { hiddenUntil: "2026-12-01T00:00:00Z" }),
  });
}

describe("Sponsored card", () => {
  it("is labelled, counts a view and a tap, and can be hidden for 30 days", async () => {
    const open = jest.spyOn(Linking, "openURL").mockResolvedValue();
    const calls = api({ card });
    await renderRouter(routes, { initialUrl: "/trips/t1/stays" });
    expect(await screen.findByText("Sponsored")).toBeTruthy();
    expect(screen.getByText("Cafe Co")).toBeTruthy();
    await waitFor(() =>
      expect(calls.some((c) => c.key === "POST /v1/sponsored/c1/impressions")).toBe(true),
    );
    await fireEvent.press(screen.getByRole("link", { name: "Sponsored: Free coffee" }));
    expect(open).toHaveBeenCalledWith("https://cafe.example.com");
    await waitFor(() =>
      expect(calls.some((c) => c.key === "POST /v1/sponsored/c1/clicks")).toBe(true),
    );
    await fireEvent.press(screen.getByRole("button", { name: "Hide for 30 days" }));
    await waitFor(() => expect(screen.queryByText("Sponsored")).toBeNull());
    expect(calls.some((c) => c.key === "POST /v1/sponsored/c1/hide")).toBe(true);
  });

  it("draws nothing when there is no card", async () => {
    api({});
    await renderRouter(routes, { initialUrl: "/trips/t1/stays" });
    await screen.findByText("No booking partners are set up yet.");
    expect(screen.queryByText("Sponsored")).toBeNull();
  });
});
