import { fireEvent, renderRouter, screen } from "expo-router/testing-library";

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

const stop = (id: string, date: string, order: string, name: string) => ({
  id,
  tripId: "t1",
  date,
  order,
  place: { name, lat: 21, lng: 105 },
  timeZone: "UTC",
});

describe("Trip map", () => {
  it("lists numbered places and filters by day", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1/stops": () =>
        json(200, {
          stops: [
            stop("a", "2099-11-19", "a", "Pho Thin"),
            stop("b", "2099-11-19", "b", "Hoan Kiem Lake"),
            stop("c", "2099-11-20", "a", "Temple of Literature"),
          ],
        }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/map" });
    expect(await screen.findByText("1. Pho Thin")).toBeTruthy();
    expect(screen.getByText("2. Hoan Kiem Lake")).toBeTruthy();
    expect(screen.getByText("1. Temple of Literature")).toBeTruthy();
    await fireEvent.press(screen.getAllByRole("button", { name: /20/ })[0] as never);
    expect(screen.queryByText("1. Pho Thin")).toBeNull();
    expect(screen.getByText("1. Temple of Literature")).toBeTruthy();
  });

  it("explains an empty plan", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1/stops": () => json(200, { stops: [] }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/map" });
    expect(await screen.findByText(/Add places to your itinerary/)).toBeTruthy();
  });
});
