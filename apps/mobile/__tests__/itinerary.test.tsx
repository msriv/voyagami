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

  it("opens Add a place for the chosen day", async () => {
    mockApi(base());
    const view = await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    await fireEvent.press(await screen.findByRole("button", { name: /^Day 2/ }));
    await fireEvent.press(screen.getByRole("button", { name: "Add a stop" }));
    expect(await screen.findByText("Add a place")).toBeTruthy();
    expect(view.getPathnameWithParams()).toBe("/trips/t1/add-place?date=2099-11-20");
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

  it("reorders with the drag handle's accessibility action", async () => {
    const calls = mockApi({
      ...base("owner", [
        stop("s1", "First", "2099-11-19", "a"),
        stop("s2", "Second", "2099-11-19", "b"),
      ]),
      "PATCH /v1/trips/t1/stops/s1": () => json(200, stop("s1", "First", "2099-11-19", "c")),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    const handle = await screen.findByLabelText("Reorder First. Drag up or down.");
    await fireEvent(handle, "accessibilityAction", { nativeEvent: { actionName: "increment" } });
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/trips/t1/stops/s1")?.body).toEqual({
        date: "2099-11-19",
      }),
    );
  });

  it("is read-only for viewers", async () => {
    mockApi(base("viewer", [stop("s1", "Old Quarter walk", "2099-11-19", "a")]));
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    expect(await screen.findByText("Old Quarter walk")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Add a stop" })).toBeNull();
    expect(screen.queryByRole("button", { name: /^Edit/ })).toBeNull();
    expect(screen.queryByLabelText(/^Reorder/)).toBeNull();
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

  it("moves a stop down, up and to another day, and puts it back if the server says no", async () => {
    let fail = false;
    const calls = mockApi({
      ...base("owner", [
        stop("s1", "First", "2099-11-19", "a"),
        stop("s2", "Second", "2099-11-19", "b"),
      ]),
      "PATCH /v1/trips/t1/stops/s1": () =>
        fail
          ? json(500, { code: "internal", message: "x" })
          : json(200, stop("s1", "First", "2099-11-19", "c")),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/itinerary" });
    const names = () =>
      screen
        .getAllByRole("button", { name: /^Edit (First|Second)$/, includeHiddenElements: true })
        .map((b) => b.props.accessibilityLabel);

    await fireEvent.press(await screen.findByRole("button", { name: "Edit First" }));
    expect(screen.getByRole("button", { name: "Move up" }).props.accessibilityState.disabled).toBe(
      true,
    );
    await fireEvent.press(screen.getByRole("button", { name: "Move down" }));
    await waitFor(() => expect(names()).toEqual(["Edit Second", "Edit First"]));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/trips/t1/stops/s1")?.body).toEqual({
        date: "2099-11-19",
      }),
    );

    fail = true;
    await fireEvent.press(screen.getByRole("button", { name: "Move up" }));
    expect(await screen.findByText("Something went wrong. Try again.")).toBeTruthy();
    await waitFor(() => expect(names()).toEqual(["Edit Second", "Edit First"]));
  });
});
