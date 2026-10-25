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

const trip = (role: string, dates: object = { start: "2099-11-19", end: "2099-11-21" }) => ({
  id: "t1",
  title: "Hanoi food week",
  ownerId: "u1",
  destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
  dates,
  timeZone: "Asia/Ho_Chi_Minh",
  settlementCurrency: "VND",
  status: "planned",
  role,
});

const stop = (id: string, name: string, date: string, order: string, extra: object = {}) => ({
  id,
  tripId: "t1",
  date,
  order,
  place: { name, lat: 21, lng: 105 },
  timeZone: "Asia/Ho_Chi_Minh",
  ...extra,
});

const base = (role = "owner", stops: object[] = []) => ({
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1": () => json(200, trip(role)),
  "GET /v1/trips/t1/stops": () => json(200, { stops }),
});

describe("Itinerary", () => {
  it("shows one day at a time, with the empty day state", async () => {
    mockApi(
      base("owner", [
        stop("s2", "Egg coffee", "2099-11-19", "b", { startLocal: "09:30", durationMin: 45 }),
        stop("s1", "Old Quarter walk", "2099-11-19", "a"),
        stop("s3", "Ha Long cruise", "2099-11-20", "a"),
      ]),
    );
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    expect(await screen.findByText("Old Quarter walk")).toBeTruthy();
    expect(screen.getByText("Egg coffee")).toBeTruthy();
    expect(screen.getByText("45 min")).toBeTruthy();
    expect(screen.queryByText("Ha Long cruise")).toBeNull();

    await fireEvent.press(screen.getByRole("button", { name: /^Day 2/ }));
    expect(screen.getByText("Ha Long cruise")).toBeTruthy();

    await fireEvent.press(screen.getByRole("button", { name: /^Day 3/ }));
    expect(screen.getByText("Nothing planned for this day yet.")).toBeTruthy();
  });

  it("adds a stop to the chosen day", async () => {
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/stops": () =>
        json(201, stop("n1", "Pho Thin", "2099-11-19", "a", { startLocal: "12:00" })),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    await fireEvent.press(await screen.findByRole("button", { name: "Add a stop" }));
    await fireEvent.changeText(screen.getByLabelText("Place name"), "Pho Thin");
    await fireEvent.changeText(screen.getByLabelText(/^Start time/), "12:00");
    await fireEvent.press(screen.getByRole("button", { name: "Add stop" }));
    expect(await screen.findByText("Pho Thin")).toBeTruthy();
    expect(calls.find((c) => c.key === "POST /v1/trips/t1/stops")?.body).toMatchObject({
      date: "2099-11-19",
      place: { name: "Pho Thin" },
      startLocal: "12:00",
    });
  });

  it("rejects a bad time before calling the server", async () => {
    const calls = mockApi(base());
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    await fireEvent.press(await screen.findByRole("button", { name: "Add a stop" }));
    await fireEvent.changeText(screen.getByLabelText("Place name"), "Pho Thin");
    await fireEvent.changeText(screen.getByLabelText(/^Start time/), "25:99");
    await fireEvent.press(screen.getByRole("button", { name: "Add stop" }));
    expect(
      await screen.findByText("Use a 24-hour time like 09:30, or leave it blank."),
    ).toBeTruthy();
    expect(calls.some((c) => c.key.startsWith("POST"))).toBe(false);
  });

  it("edits and removes a stop", async () => {
    const calls = mockApi({
      ...base("editor", [stop("s1", "Old Quarter walk", "2099-11-19", "a")]),
      "PATCH /v1/trips/t1/stops/s1": () =>
        json(200, stop("s1", "Old Quarter walk", "2099-11-19", "a", { startLocal: "08:00" })),
      "DELETE /v1/trips/t1/stops/s1": () => json(204),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    await fireEvent.press(await screen.findByRole("button", { name: "Edit Old Quarter walk" }));
    await fireEvent.changeText(screen.getByLabelText(/^Start time/), "08:00");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/trips/t1/stops/s1")?.body).toEqual({
        startLocal: "08:00",
        durationMin: 0,
        note: "",
      }),
    );

    await fireEvent.press(await screen.findByRole("button", { name: "Edit Old Quarter walk" }));
    await fireEvent.press(screen.getByRole("button", { name: "Remove this stop" }));
    await waitFor(() => expect(screen.queryByText("Old Quarter walk")).toBeNull());
  });

  it("is read-only for viewers", async () => {
    mockApi(base("viewer", [stop("s1", "Old Quarter walk", "2099-11-19", "a")]));
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    expect(await screen.findByText("Old Quarter walk")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Add a stop" })).toBeNull();
    expect(screen.queryByRole("button", { name: /^Edit/ })).toBeNull();
  });

  it("asks for dates on a trip without fixed days", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1": () => json(200, trip("owner", { flexible: { days: 3 } })),
      "GET /v1/trips/t1/stops": () => json(200, { stops: [] }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    expect(await screen.findByText("Set dates for this trip to plan it day by day.")).toBeTruthy();
  });
});
