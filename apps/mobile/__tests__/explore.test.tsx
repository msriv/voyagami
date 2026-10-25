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
  mockApi({
    "GET /v1/me": () => json(200, { id: "u1" }),
    "GET /v1/trips": () => json(200, { trips: [] }),
  });
});

describe("Explore tab", () => {
  it("filters places by name or country", async () => {
    await renderRouter(routes, { initialUrl: "/explore" });
    expect(await screen.findByRole("button", { name: "Plan a trip to Hanoi" })).toBeTruthy();
    await fireEvent.changeText(screen.getByLabelText("Search places"), "japan");
    expect(screen.getByRole("button", { name: "Plan a trip to Kyoto" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Plan a trip to Hanoi" })).toBeNull();
    await fireEvent.changeText(screen.getByLabelText("Search places"), "zzz");
    expect(screen.getByText("Nothing matches that yet.")).toBeTruthy();
  });

  it("starts a new trip with the place already picked", async () => {
    await renderRouter(routes, { initialUrl: "/explore" });
    await fireEvent.press(await screen.findByRole("button", { name: "Plan a trip to Hanoi" }));
    expect(await screen.findByRole("button", { name: "Remove Hanoi" })).toBeTruthy();
  });
});
