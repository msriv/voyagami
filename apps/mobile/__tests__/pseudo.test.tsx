import { renderRouter, screen } from "expo-router/testing-library";

import { i18n } from "../src/i18n";
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

afterEach(async () => {
  await i18n.changeLanguage("en");
});

describe("Pseudo-locale smoke", () => {
  it("renders Home with every string longer and accented, and still shows the profile button", async () => {
    mockStore.clear();
    mockStore.set("voyagami.accessToken", "a");
    mockStore.set("voyagami.refreshToken", "r");
    mockStore.set("voyagami.onboarded", "1");
    mockStore.set("voyagami.tour", "done");
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips": () => json(200, { trips: [] }),
    });
    await i18n.changeLanguage("en-XA");
    await renderRouter(routes, { initialUrl: "/" });
    expect(await screen.findByText(/^\[Ýöür/)).toBeTruthy();
    expect(screen.getByRole("button", { name: /^\[Öpéñ/ })).toBeTruthy();
  });
});
