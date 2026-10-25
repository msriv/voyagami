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

const trip = (role: string, status = "planned") => ({
  id: "t1",
  title: "Hanoi food week",
  ownerId: "u1",
  destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
  dates: { start: "2099-11-19", end: "2099-11-21" },
  timeZone: "Asia/Ho_Chi_Minh",
  settlementCurrency: "VND",
  status,
  role,
});

describe("Trip overview", () => {
  it("shows the trip, its days and the section buttons", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1": () => json(200, trip("owner")),
      "GET /v1/trips/t1/tasks": () => json(200, { tasks: [] }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1" });
    expect(await screen.findByText("Hanoi food week")).toBeTruthy();
    expect(screen.getByText("Your role: Owner")).toBeTruthy();
    expect(screen.getAllByLabelText(/^Day \d/)).toHaveLength(3);
    expect(screen.getByRole("button", { name: "Journal" })).toBeTruthy();
  });

  it("says when the trip cannot be found", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1": () => json(404, { code: "not_found", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1" });
    expect(await screen.findByText("We couldn't find that trip.")).toBeTruthy();
  });

  it("lets an owner archive the trip", async () => {
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1": () => json(200, trip("owner")),
      "GET /v1/trips/t1/tasks": () => json(200, { tasks: [] }),
      "PATCH /v1/trips/t1": () => json(200, trip("owner", "past")),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1" });
    await fireEvent.press(await screen.findByRole("button", { name: "Trip menu" }));
    await fireEvent.press(await screen.findByRole("button", { name: "Archive trip" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/trips/t1")?.body).toEqual({ status: "past" }),
    );
    await waitFor(() => screen.getByText("ARCHIVED", { includeHiddenElements: true }));
  });

  it("shows viewers a read-only menu", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u2" }),
      "GET /v1/trips/t1": () => json(200, trip("viewer")),
      "GET /v1/trips/t1/tasks": () => json(200, { tasks: [] }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1" });
    await fireEvent.press(await screen.findByRole("button", { name: "Trip menu" }));
    expect(
      await screen.findByText("Only the owner and editors can change this trip."),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Archive trip" })).toBeNull();
  });
});
