import { renderRouter, screen } from "expo-router/testing-library";

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
});

const trip = {
  id: "t1",
  title: "Vietnam",
  dates: { start: "2026-11-19", end: "2026-11-27" },
};
const recap = {
  tripId: "t1",
  days: 9,
  stops: 34,
  crew: 4,
  distanceMeters: 12500,
  spentMinor: 1482000,
  currency: "VND",
  remainingMinor: 118000,
  settledUp: true,
  favouriteDay: { date: "2026-11-21", entries: 2 },
  highlights: [{ date: "2026-11-21", authorUid: "u2", text: "Fog on the bay." }],
};

const api = (r: object) => ({
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1": () => json(200, trip),
  "GET /v1/trips/t1/members": () =>
    json(200, { members: [{ uid: "u2", role: "editor", status: "active", displayName: "Ben" }] }),
  "GET /v1/trips/t1/recap": () => json(200, r),
});

describe("Trip recap", () => {
  it("shows the stamp, numbers, money and highlights", async () => {
    mockApi(api(recap));
    await renderRouter(routes, { initialUrl: "/trips/t1/recap" });
    expect(await screen.findByText("Vietnam")).toBeTruthy();
    expect(screen.getByText("9 days")).toBeTruthy();
    expect(screen.getByText("34 stops")).toBeTruthy();
    expect(screen.getByText("4 travellers")).toBeTruthy();
    expect(screen.getByText("Everyone settled up")).toBeTruthy();
    expect(screen.getByText("Fog on the bay.")).toBeTruthy();
    expect(screen.getByText(/Ben/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Plan the next one" })).toBeTruthy();
  });

  it("says what is missing for a trip with nothing in it yet", async () => {
    mockApi(
      api({
        ...recap,
        stops: 0,
        spentMinor: 0,
        highlights: [],
        favouriteDay: undefined,
        remainingMinor: undefined,
        settledUp: undefined,
        distanceMeters: 0,
      }),
    );
    await renderRouter(routes, { initialUrl: "/trips/t1/recap" });
    expect(await screen.findByText(/your recap will fill in here/)).toBeTruthy();
  });

  it("offers a retry when the recap cannot load", async () => {
    mockApi({
      ...api(recap),
      "GET /v1/trips/t1/recap": () => json(500, { code: "internal", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/recap" });
    expect(await screen.findByRole("button", { name: "Try again" })).toBeTruthy();
  });
});
