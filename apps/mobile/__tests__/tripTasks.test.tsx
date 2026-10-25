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

describe("Trip to-dos", () => {
  it("lists, ticks, adds and deletes", async () => {
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1": () => json(200, trip("owner")),
      "GET /v1/trips/t1/tasks": () =>
        json(200, { tasks: [{ id: "k1", title: "Book SIM", done: false }] }),
      "PATCH /v1/trips/t1/tasks/k1": () => json(200, { id: "k1", title: "Book SIM", done: true }),
      "POST /v1/trips/t1/tasks": () => json(201, { id: "k2", title: "Buy adapter", done: false }),
      "DELETE /v1/trips/t1/tasks/k1": () => json(204),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1" });
    const box = await screen.findByRole("checkbox", { name: "Book SIM" });
    expect(box.props.accessibilityState.checked).toBe(false);

    await fireEvent.press(box);
    expect(
      screen.getByRole("checkbox", { name: "Book SIM" }).props.accessibilityState.checked,
    ).toBe(true);
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/trips/t1/tasks/k1")?.body).toEqual({
        done: true,
      }),
    );

    await fireEvent.changeText(screen.getByLabelText("New to-do"), "Buy adapter");
    await fireEvent.press(screen.getByRole("button", { name: "Add to-do" }));
    expect(await screen.findByRole("checkbox", { name: "Buy adapter" })).toBeTruthy();
    expect(calls.find((c) => c.key === "POST /v1/trips/t1/tasks")?.body).toEqual({
      title: "Buy adapter",
    });

    await fireEvent.press(screen.getByRole("button", { name: "Delete Book SIM" }));
    expect(screen.queryByRole("checkbox", { name: "Book SIM" })).toBeNull();
  });

  it("puts a tick back and says why when the server says no", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips/t1": () => json(200, trip("owner")),
      "GET /v1/trips/t1/tasks": () =>
        json(200, { tasks: [{ id: "k1", title: "Book SIM", done: false }] }),
      "PATCH /v1/trips/t1/tasks/k1": () => json(500, { code: "internal", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1" });
    await fireEvent.press(await screen.findByRole("checkbox", { name: "Book SIM" }));
    expect(await screen.findByText("Something went wrong. Try again.")).toBeTruthy();
    expect(
      screen.getByRole("checkbox", { name: "Book SIM" }).props.accessibilityState.checked,
    ).toBe(false);
  });

  it("is read-only for viewers", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u2" }),
      "GET /v1/trips/t1": () => json(200, trip("viewer")),
      "GET /v1/trips/t1/tasks": () =>
        json(200, { tasks: [{ id: "k1", title: "Book SIM", done: false }] }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1" });
    const box = await screen.findByRole("checkbox", { name: "Book SIM" });
    expect(box.props.accessibilityState.disabled).toBe(true);
    expect(screen.queryByLabelText("New to-do")).toBeNull();
    expect(screen.queryByRole("button", { name: "Delete Book SIM" })).toBeNull();
  });
});
