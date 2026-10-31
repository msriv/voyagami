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

const item = {
  placeId: "p1",
  name: "Water puppet show",
  distanceMeters: 800,
  links: [{ partnerId: "k", name: "Klook", url: "https://klook.example.com/p1" }],
};

describe("Things to do", () => {
  it("opens a partner and counts the tap, and adds the place at the typed time", async () => {
    const open = jest.spyOn(Linking, "openURL").mockResolvedValue();
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1/experiences": () => json(200, { items: [item] }),
      "POST /v1/trips/t1/experiences/clicks": () => new Response(null, { status: 204 }),
      "GET /v1/trips/t1": () =>
        json(200, { id: "t1", dates: { start: "2026-12-01", end: "2026-12-04" } }),
      "POST /v1/trips/t1/stops": () => json(200, { id: "s1" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/experiences" });
    await fireEvent.press(await screen.findByRole("button", { name: "Book on Klook" }));
    expect(open).toHaveBeenCalledWith("https://klook.example.com/p1");
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/experiences/clicks")?.body).toEqual({
        partnerId: "k",
      }),
    );
    await fireEvent.press(
      screen.getByRole("button", { name: "Add Water puppet show to itinerary" }),
    );
    await fireEvent.changeText(screen.getByLabelText("Start time (24-hour, like 14:30)"), "25:99");
    await fireEvent.press(screen.getByRole("button", { name: "Add to day 1" }));
    expect(await screen.findByText(/Use a 24-hour time/)).toBeTruthy();
    await fireEvent.changeText(screen.getByLabelText("Start time (24-hour, like 14:30)"), "19:30");
    await fireEvent.press(screen.getByRole("button", { name: "Add to day 1" }));
    expect(await screen.findByText("Added Water puppet show to your trip.")).toBeTruthy();
    expect(calls.find((c) => c.key === "POST /v1/trips/t1/stops")?.body).toEqual({
      date: "2026-12-01",
      place: { placeId: "p1" },
      startLocal: "19:30",
    });
  });

  it("says so when the list is switched off", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1/experiences": () => json(503, { code: "unavailable", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/experiences" });
    expect(await screen.findByText("Things to do are not available yet.")).toBeTruthy();
  });
});
