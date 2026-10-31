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
  mockStore.set("voyagami.tour", "done");
});

const draft = {
  days: [
    { day: 1, items: [{ placeId: "p1", name: "Old Quarter walk", note: "Easy and central" }] },
    { day: 2, items: [{ placeId: "p2", name: "Egg coffee stop" }] },
  ],
  remaining: 2,
};

describe("Ask", () => {
  it("makes a draft, lets the person untick a place, and adds the rest on the right days", async () => {
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "POST /v1/trips/t1/ask/drafts": () => json(200, draft),
      "GET /v1/trips/t1": () =>
        json(200, { id: "t1", dates: { start: "2026-12-01", end: "2026-12-05" } }),
      "POST /v1/trips/t1/stops": () => json(200, { id: "s1" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/ask" });
    await fireEvent.changeText(
      await screen.findByLabelText("What do you want to do?"),
      "food trip",
    );
    await fireEvent.press(screen.getByRole("button", { name: "Make a draft" }));
    expect(await screen.findByText("Easy and central")).toBeTruthy();
    expect(screen.getByText("2 drafts left this month.")).toBeTruthy();
    await fireEvent.press(screen.getByRole("checkbox", { name: "Egg coffee stop" }));
    await fireEvent.press(screen.getByRole("button", { name: "Add 1 to my trip" }));
    expect(await screen.findByText("Added 1 place to your trip.")).toBeTruthy();
    const stops = calls.filter((c) => c.key === "POST /v1/trips/t1/stops");
    expect(stops.map((c) => c.body)).toEqual([{ date: "2026-12-01", place: { placeId: "p1" } }]);
  });

  it("shows the paywall when the monthly limit is used up", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "POST /v1/trips/t1/ask/drafts": () =>
        json(403, { code: "limit_reached", feature: "askDrafts", message: "limit" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/ask" });
    await fireEvent.changeText(
      await screen.findByLabelText("What do you want to do?"),
      "food trip",
    );
    await fireEvent.press(screen.getByRole("button", { name: "Make a draft" }));
    expect(await screen.findByText("More Ask drafts this month")).toBeTruthy();
  });
});
