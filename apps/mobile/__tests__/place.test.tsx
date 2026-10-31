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

const place = {
  id: "p1",
  name: "Hoan Kiem Lake",
  lat: 21.028,
  lng: 105.852,
  address: "Hang Trong",
  group: "see",
  categories: [],
  photos: [
    {
      url: "https://img.example.com/a.jpg",
      attribution: "Ann on Wikimedia",
      source: "w",
      licence: "CC",
    },
  ],
  sources: [
    { field: "name", source: "OpenStreetMap", licence: "ODbL" },
    { field: "photo", source: "Wikimedia", licence: "CC" },
  ],
};

describe("Place detail", () => {
  it("shows credits and sources, opens the map and adds to the trip", async () => {
    const open = jest.spyOn(Linking, "openURL").mockResolvedValue();
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/places/p1": () => json(200, place),
      "POST /v1/trips/t1/stops": () => json(201, { id: "s1" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/place?placeId=p1&date=2099-11-19" });
    expect(await screen.findByText("Photo: Ann on Wikimedia")).toBeTruthy();
    expect(screen.getByText("Details from OpenStreetMap, Wikimedia.")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Open in maps" }));
    expect(open).toHaveBeenCalledWith(expect.stringContaining("21.028,105.852"));
    await fireEvent.press(screen.getByRole("button", { name: "Add to the trip" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/stops")?.body).toMatchObject({
        date: "2099-11-19",
        place: { name: "Hoan Kiem Lake", category: "sights" },
      }),
    );
    expect(await screen.findByText("Added to the trip.")).toBeTruthy();
  });

  it("says when the place is gone", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/places/p1": () => json(404, { code: "not_found", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/place?placeId=p1" });
    expect(await screen.findByText("We couldn't find this place.")).toBeTruthy();
  });
});
