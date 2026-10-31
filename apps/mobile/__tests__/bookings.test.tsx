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

const base = (role = "owner", bookings: object[] = []) => ({
  "GET /v1/trips/t1": () => json(200, trip(role)),
  "GET /v1/trips/t1/bookings": () => json(200, { bookings }),
});

describe("Bookings", () => {
  it("lists bookings soonest first, undated last, with local times", async () => {
    mockApi(
      base("owner", [
        { id: "b3", kind: "stay", title: "Old Quarter hotel" },
        {
          id: "b2",
          kind: "train",
          title: "Night train",
          startsAt: "2099-11-20T12:00:00Z",
          startTimeZone: "Asia/Ho_Chi_Minh",
        },
        {
          id: "b1",
          kind: "flight",
          title: "VN 123",
          provider: "Vietnam Airlines",
          startsAt: "2099-11-19T02:30:00Z",
          startTimeZone: "Asia/Ho_Chi_Minh",
        },
      ]),
    );
    await renderRouter(routes, { initialUrl: "/trips/t1/bookings" });
    expect(await screen.findByText("VN 123")).toBeTruthy();
    expect(screen.getByText("Vietnam Airlines")).toBeTruthy();
    expect(screen.getByText("No time set")).toBeTruthy();
    expect(screen.getByText(/9:30/)).toBeTruthy();
    const names = screen
      .getAllByRole("button", { name: /^(Flight|Train|Stay),/ })
      .map((b) => b.props.accessibilityLabel.split(",")[1].trim());
    expect(names).toEqual(["VN 123", "Night train", "Old Quarter hotel"]);
  });

  it("shows the empty state and hides Add for viewers", async () => {
    mockApi(base("viewer"));
    await renderRouter(routes, { initialUrl: "/trips/t1/bookings" });
    expect(await screen.findByText("No bookings yet.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Add a booking" })).toBeNull();
  });

  it("deletes a booking", async () => {
    const calls = mockApi({
      ...base("owner", [{ id: "b1", kind: "flight", title: "VN 123" }]),
      "DELETE /v1/trips/t1/bookings/b1": () => json(204),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/bookings" });
    await fireEvent.press(await screen.findByRole("button", { name: /^Flight, VN 123/ }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete this booking" }));
    await waitFor(() =>
      expect(calls.some((c) => c.key === "DELETE /v1/trips/t1/bookings/b1")).toBe(true),
    );
    await waitFor(() => expect(screen.queryByText("VN 123")).toBeNull());
  });
});

describe("Add booking", () => {
  it("sends the start as a UTC instant in the trip's zone", async () => {
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/bookings": () => json(201, { id: "b1", kind: "flight", title: "VN 123" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/add-booking" });
    await fireEvent.changeText(await screen.findByLabelText("Name"), "VN 123");
    await fireEvent.changeText(
      screen.getByLabelText("Airline, hotel or company"),
      "Vietnam Airlines",
    );
    await fireEvent.press(await screen.findByRole("button", { name: "Day 2" }));
    await fireEvent.changeText(screen.getByLabelText(/^Start time/), "09:30");
    await fireEvent.press(screen.getByRole("button", { name: "Save booking" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/bookings")?.body).toEqual({
        kind: "flight",
        title: "VN 123",
        provider: "Vietnam Airlines",
        startsAt: "2099-11-20T02:30:00.000Z",
        startTimeZone: "Asia/Ho_Chi_Minh",
      }),
    );
  });

  it("asks for a name, and a day when there is a time", async () => {
    mockApi(base());
    await renderRouter(routes, { initialUrl: "/trips/t1/add-booking" });
    await fireEvent.press(await screen.findByRole("button", { name: "Save booking" }));
    expect(await screen.findByText("Give the booking a name.")).toBeTruthy();
    await fireEvent.changeText(screen.getByLabelText("Name"), "VN 123");
    await fireEvent.changeText(screen.getByLabelText(/^Start time/), "09:30");
    await fireEvent.press(screen.getByRole("button", { name: "Save booking" }));
    expect(await screen.findByText("Pick the day it starts.")).toBeTruthy();
  });

  it("checks a flight's status, and says so when no provider is set up", async () => {
    let ok = true;
    mockApi({
      ...base("owner", [{ id: "b1", kind: "flight", title: "VN 123" }]),
      "GET /v1/trips/t1/bookings/b1/flight-status": () =>
        ok
          ? json(200, {
              state: "delayed",
              departure: {
                scheduled: "2099-11-19T02:30:00Z",
                estimated: "2099-11-19T03:15:00Z",
                gate: "B4",
              },
            })
          : json(503, { code: "unavailable", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/bookings" });
    await fireEvent.press(await screen.findByRole("button", { name: "Check flight status" }));
    expect(await screen.findByText("Delayed")).toBeTruthy();
    expect(screen.getByText(/Gate B4/)).toBeTruthy();
    ok = false;
    await fireEvent.press(screen.getByRole("button", { name: "Check flight status" }));
    expect(await screen.findByText("Live flight status is not available yet.")).toBeTruthy();
  });
});
