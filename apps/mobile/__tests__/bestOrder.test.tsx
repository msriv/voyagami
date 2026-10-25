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
});

const trip = (role: string) => ({
  id: "t1",
  title: "Hanoi food week",
  ownerId: "u1",
  destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
  dates: { start: "2099-11-19", end: "2099-11-21" },
  timeZone: "Asia/Ho_Chi_Minh",
  settlementCurrency: "VND",
  status: "planned",
  role,
});

const stop = (id: string, order: string) => ({
  id,
  tripId: "t1",
  date: "2099-11-19",
  order,
  place: { name: id.toUpperCase(), lat: 21, lng: 105 },
  timeZone: "Asia/Ho_Chi_Minh",
});

const stops = [stop("a", "a"), stop("b", "b"), stop("c", "c")];

const base = (role = "owner", list = stops) => ({
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1": () => json(200, trip(role)),
  "GET /v1/trips/t1/stops": () => json(200, { stops: list }),
});

const suggestion = (changed: boolean, stopIds: string[]) => ({
  date: "2099-11-19",
  changed,
  approx: true,
  beforeMin: 50,
  afterMin: 30,
  stopIds,
});

const order = () =>
  screen
    .getAllByRole("button", { name: /^Edit [ABC]$/, includeHiddenElements: true })
    .map((b) => b.props.accessibilityLabel);

describe("Best order", () => {
  it("previews the suggestion, applies it and undoes it", async () => {
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/days/2099-11-19/best-order": () =>
        json(200, suggestion(true, ["c", "a", "b"])),
      "PUT /v1/trips/t1/days/2099-11-19/order": () => json(200, { stops }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    await fireEvent.press(await screen.findByRole("button", { name: "Suggest the best order" }));
    expect(
      await screen.findByText(
        "Travel time: about 50 min now, about 30 min in the suggested order.",
      ),
    ).toBeTruthy();

    await fireEvent.press(screen.getByRole("button", { name: "Use this order" }));
    await waitFor(() => expect(order()).toEqual(["Edit C", "Edit A", "Edit B"]));
    await waitFor(() =>
      expect(calls.filter((c) => c.key.startsWith("PUT"))[0]?.body).toEqual({
        stopIds: ["c", "a", "b"],
      }),
    );

    await fireEvent.press(await screen.findByRole("button", { name: "Undo" }));
    await waitFor(() => expect(order()).toEqual(["Edit A", "Edit B", "Edit C"]));
    await waitFor(() =>
      expect(calls.filter((c) => c.key.startsWith("PUT"))[1]?.body).toEqual({
        stopIds: ["a", "b", "c"],
      }),
    );
  });

  it("says when the order is already the best", async () => {
    mockApi({
      ...base(),
      "POST /v1/trips/t1/days/2099-11-19/best-order": () =>
        json(200, suggestion(false, ["a", "b", "c"])),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    await fireEvent.press(await screen.findByRole("button", { name: "Suggest the best order" }));
    expect(await screen.findByText("This order is already the shortest we can find.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Use this order" })).toBeNull();
  });

  it("puts the old order back if the server refuses", async () => {
    mockApi({
      ...base(),
      "POST /v1/trips/t1/days/2099-11-19/best-order": () =>
        json(200, suggestion(true, ["c", "a", "b"])),
      "PUT /v1/trips/t1/days/2099-11-19/order": () => json(500, { code: "internal", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    await fireEvent.press(await screen.findByRole("button", { name: "Suggest the best order" }));
    await fireEvent.press(await screen.findByRole("button", { name: "Use this order" }));
    expect(await screen.findByText("Something went wrong. Try again.")).toBeTruthy();
    expect(order()).toEqual(["Edit A", "Edit B", "Edit C"]);
  });

  it("is hidden for viewers", async () => {
    mockApi(base("viewer"));
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    await screen.findByText("A");
    expect(screen.queryByRole("button", { name: "Suggest the best order" })).toBeNull();
  });
});
