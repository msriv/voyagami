import { renderRouter, screen } from "expo-router/testing-library";

import { localStore } from "../src/local/store";
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
  title: "Hanoi food week",
  ownerId: "u1",
  destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
  dates: { start: "2099-11-19", end: "2099-11-21" },
  timeZone: "Asia/Ho_Chi_Minh",
  settlementCurrency: "VND",
  status: "planned",
  role: "owner",
};
const stop = {
  id: "s1",
  tripId: "t1",
  date: "2099-11-19",
  order: "a",
  place: { name: "Pho Thin" },
  timeZone: "Asia/Ho_Chi_Minh",
};

describe("Offline", () => {
  it("shows the saved itinerary when the network is down", async () => {
    await localStore().put("stops:t1", { trip, stops: [stop] });
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1": () => {
        throw new TypeError("Network request failed");
      },
      "GET /v1/trips/t1/stops": () => {
        throw new TypeError("Network request failed");
      },
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    expect(await screen.findByText("Pho Thin")).toBeTruthy();
  });

  it("shows the saved trips list when the network is down", async () => {
    await localStore().put("trips", [trip]);
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips": () => {
        throw new TypeError("Network request failed");
      },
    });
    await renderRouter(routes, { initialUrl: "/" });
    expect(await screen.findByText("Hanoi food week")).toBeTruthy();
  });
});
