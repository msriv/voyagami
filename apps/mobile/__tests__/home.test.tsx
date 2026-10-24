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

const trip = (id: string, title: string, dates: object, status = "planned") => ({
  id,
  title,
  ownerId: "u1",
  destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
  dates,
  timeZone: "Asia/Ho_Chi_Minh",
  settlementCurrency: "VND",
  status,
  role: "owner",
});

describe("Home", () => {
  it("shows the empty state with a way to plan a trip", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips": () => json(200, { trips: [] }),
    });
    await renderRouter(routes);
    expect(await screen.findByText("Your passport is empty. For now.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "New trip" })).toBeTruthy();
  });

  it("shows the next trip big, others under Upcoming, and old ones under Past trips", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips": () =>
        json(200, {
          trips: [
            trip("a", "Hanoi food week", { start: "2099-11-19", end: "2099-11-27" }),
            trip("b", "Weekend away", { flexible: { days: 3 } }),
            trip("c", "Old trip", { start: "2020-01-01", end: "2020-01-05" }),
            trip("d", "Archived", { start: "2099-02-01", end: "2099-02-02" }, "past"),
          ],
        }),
    });
    await renderRouter(routes);
    expect(await screen.findByText("Hanoi food week")).toBeTruthy();
    expect(screen.getByText("Upcoming")).toBeTruthy();
    expect(screen.getByText("Weekend away")).toBeTruthy();
    expect(screen.getByText("Past trips")).toBeTruthy();
    expect(screen.getByText("Old trip")).toBeTruthy();
    expect(screen.getByText("Archived")).toBeTruthy();
    expect(screen.getByText("3 days, dates not fixed", { exact: false })).toBeTruthy();
  });

  it("offers a retry when trips cannot be loaded", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips": () => json(503, { code: "unavailable", message: "x" }),
    });
    await renderRouter(routes);
    expect(await screen.findByText("We couldn't load your trips.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
  });
});
