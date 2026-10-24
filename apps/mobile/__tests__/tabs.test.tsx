import { renderRouter, screen } from "expo-router/testing-library";

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

describe("tab shell", () => {
  it("shows Home, Explore and Profile tabs when signed in", async () => {
    mockStore.set("voyagami.accessToken", "a");
    mockStore.set("voyagami.refreshToken", "r");
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips": () => json(200, { trips: [] }),
    });
    await renderRouter(routes);
    expect(await screen.findByRole("button", { name: "Home" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Explore" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Profile" })).toBeTruthy();
  });
});
