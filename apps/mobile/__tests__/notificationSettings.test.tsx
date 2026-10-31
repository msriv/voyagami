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
});

describe("Notification settings", () => {
  it("saves a switched-off kind and rolls back when saving fails", async () => {
    let fail = false;
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1", notifications: { pollCreated: false } }),
      "PATCH /v1/me": () =>
        fail ? json(500, { code: "internal", message: "no" }) : json(200, { id: "u1" }),
    });
    await renderRouter(routes, { initialUrl: "/notifications" });
    const polls = await screen.findByRole("button", { name: "Someone starts a poll: Off" });
    expect(polls.props.accessibilityState.selected).toBe(false);

    await fireEvent.press(screen.getByRole("button", { name: "Someone adds a place: On" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/me")?.body).toEqual({
        notifications: { pollCreated: false, stopAdded: false },
      }),
    );

    fail = true;
    await fireEvent.press(screen.getByRole("button", { name: "Someone adds a place: Off" }));
    expect(await screen.findByText(/couldn't save/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Someone adds a place: Off" })).toBeTruthy();
  });
});
