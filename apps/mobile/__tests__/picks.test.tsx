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

const pick = {
  place: {
    id: "p1",
    name: "Bun Cha Huong Lien",
    lat: 21.01,
    lng: 105.85,
    address: "24 Le Van Huu",
    group: "eat",
    categories: [],
    sources: [],
    photos: [
      {
        url: "https://img.example.com/a.jpg",
        attribution: "Ann on Wikimedia",
        source: "w",
        licence: "CC",
      },
    ],
  },
  score: 0.9,
  reasons: ["style", "distance"],
};

const base = (hasMore: boolean, plan = "free") => ({
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1/entitlements": () => json(200, { plan, features: {} }),
  "GET /v1/trips/t1/recommendations": () => json(200, { category: "eat", items: [pick], hasMore }),
});

describe("Picks for you", () => {
  it("shows the pick with reasons and photo credit, and adds it to the trip", async () => {
    const calls = mockApi({
      ...base(false),
      "GET /v1/trips/t1": () =>
        json(200, { id: "t1", dates: { start: "2099-11-19", end: "2099-11-21" } }),
      "POST /v1/trips/t1/stops": () => json(201, { id: "s1" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/picks" });
    expect(await screen.findByText("Bun Cha Huong Lien")).toBeTruthy();
    expect(screen.getByText("Fits your crew's style")).toBeTruthy();
    expect(screen.getByText("Photo: Ann on Wikimedia")).toBeTruthy();
    await fireEvent.press(
      screen.getByRole("button", { name: "Add Bun Cha Huong Lien to the trip" }),
    );
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/stops")?.body).toMatchObject({
        date: "2099-11-19",
        place: { name: "Bun Cha Huong Lien", category: "food" },
      }),
    );
    expect(await screen.findByText("Added Bun Cha Huong Lien to the trip.")).toBeTruthy();
  });

  it("opens the paywall from See more on Free", async () => {
    mockApi(base(true));
    await renderRouter(routes, { initialUrl: "/trips/t1/picks" });
    await fireEvent.press(await screen.findByRole("button", { name: "See more picks" }));
    expect(await screen.findByText("Try Plus free for 7 days")).toBeTruthy();
  });

  it("says when the trip has no pinned stop", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1/recommendations": () => json(400, { code: "bad_request", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/picks" });
    expect(await screen.findByText(/Add a stop with a pin/)).toBeTruthy();
  });
});
