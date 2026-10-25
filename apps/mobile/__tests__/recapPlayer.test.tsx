import { act, fireEvent, renderRouter, screen } from "expo-router/testing-library";
import { AccessibilityInfo } from "react-native";

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
  jest.spyOn(AccessibilityInfo, "isReduceMotionEnabled").mockResolvedValue(false);
  mockApi({
    "GET /v1/me": () => json(200, { id: "u1" }),
    "GET /v1/trips/t1": () =>
      json(200, { id: "t1", title: "Vietnam", dates: { start: "2026-11-19", end: "2026-11-27" } }),
    "GET /v1/trips/t1/members": () => json(200, { members: [] }),
    "GET /v1/trips/t1/recap": () =>
      json(200, {
        tripId: "t1",
        days: 9,
        stops: 34,
        crew: 4,
        distanceMeters: 0,
        spentMinor: 0,
        currency: "VND",
        highlights: [],
      }),
  });
});

afterEach(() => {
  jest.useRealTimers();
  jest.restoreAllMocks();
});

describe("Recap player", () => {
  it("steps through the cards with the buttons and ends on a closing card", async () => {
    await renderRouter(routes, { initialUrl: "/trips/t1/recap-player" });
    expect(await screen.findByText("Vietnam")).toBeTruthy();
    expect(screen.getByLabelText("Card 1 of 3")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Previous card" })).toBeDisabled();

    await fireEvent.press(screen.getByRole("button", { name: "Next card" }));
    expect(screen.getByText("9 days")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Next card" }));
    expect(screen.getByText("That was the trip.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Next card" })).toBeDisabled();

    await fireEvent.press(screen.getByRole("button", { name: "Watch again" }));
    expect(screen.getByText("Vietnam")).toBeTruthy();
  });

  it("moves on by itself, and stops when paused", async () => {
    await renderRouter(routes, { initialUrl: "/trips/t1/recap-player" });
    await screen.findByText("Vietnam");
    jest.useFakeTimers();
    // Moving a card starts a fresh timer, now on the fake clock.
    await fireEvent.press(screen.getByRole("button", { name: "Next card" }));
    expect(screen.getByText("9 days")).toBeTruthy();
    await act(async () => {
      jest.advanceTimersByTime(5000);
    });
    expect(screen.getByText("That was the trip.")).toBeTruthy();
  });

  it("stays on a card while paused", async () => {
    await renderRouter(routes, { initialUrl: "/trips/t1/recap-player" });
    await screen.findByText("Vietnam");
    jest.useFakeTimers();
    await fireEvent.press(screen.getByRole("button", { name: "Next card" }));
    await fireEvent.press(screen.getByRole("button", { name: "Pause" }));
    await act(async () => {
      jest.advanceTimersByTime(20000);
    });
    expect(screen.getByText("9 days")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Play" })).toBeTruthy();
  });

  it("does not move on by itself when reduce motion is on", async () => {
    jest.spyOn(AccessibilityInfo, "isReduceMotionEnabled").mockResolvedValue(true);
    await renderRouter(routes, { initialUrl: "/trips/t1/recap-player" });
    await screen.findByText("Vietnam");
    jest.useFakeTimers();
    await act(async () => {
      jest.advanceTimersByTime(30000);
    });
    expect(screen.getByText("Vietnam")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Pause" })).toBeNull();
    await fireEvent.press(screen.getByRole("button", { name: "Next card" }));
    expect(screen.getByText("9 days")).toBeTruthy();
  });
});
