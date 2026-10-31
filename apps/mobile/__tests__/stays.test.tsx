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

describe("Stays", () => {
  it("opens a partner, counts the tap, and says booking happens there", async () => {
    const open = jest.spyOn(Linking, "openURL").mockResolvedValue();
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1/stays/links": () =>
        json(200, {
          links: [
            { partnerId: "p1", name: "Example Stays", url: "https://stays.example.com/?q=Hanoi" },
          ],
        }),
      "POST /v1/trips/t1/stays/clicks": () => new Response(null, { status: 204 }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/stays" });
    expect(await screen.findByText(/Booking happens on the partner's site/)).toBeTruthy();
    await fireEvent.press(await screen.findByRole("button", { name: "Search on Example Stays" }));
    expect(open).toHaveBeenCalledWith("https://stays.example.com/?q=Hanoi");
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/stays/clicks")?.body).toEqual({
        partnerId: "p1",
      }),
    );
    expect(screen.getByRole("button", { name: "Add the stay to Bookings" })).toBeTruthy();
  });

  it("says so when no partner is set up or the trip has no destination", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1/stays/links": () => json(200, { links: [] }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/stays" });
    expect(await screen.findByText("No booking partners are set up yet.")).toBeTruthy();
  });
});
