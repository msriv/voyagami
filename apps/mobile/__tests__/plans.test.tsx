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
  mockPrices = {};
  mockStore.clear();
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
  mockStore.set("voyagami.onboarded", "1");
});

let mockPrices: object = {};
jest.mock("../src/plans/prices", () => ({ usePlanPrices: () => mockPrices }));

const entitlements = (plan: string) => ({
  "GET /v1/me/entitlements": () => json(200, { plan, features: {} }),
});

describe("Plans", () => {
  it("shows three plans, marks the current one and never invents a price", async () => {
    mockApi(entitlements("plus"));
    await renderRouter(routes, { initialUrl: "/plans" });
    expect(await screen.findByText("Voyagami Plus")).toBeTruthy();
    expect(await screen.findByText("YOUR PLAN")).toBeTruthy();
    expect(screen.getAllByText("Price shown by the store").length).toBeGreaterThanOrEqual(4);
    expect(screen.getByText("7 days free")).toBeTruthy();
    expect(screen.getByLabelText("Weekend, up to 4 days: Price shown by the store")).toBeTruthy();
  });

  it("says restore is not switched on when purchases are not set up", async () => {
    mockApi(entitlements("free"));
    await renderRouter(routes, { initialUrl: "/plans" });
    await fireEvent.press(await screen.findByRole("button", { name: "Restore purchases" }));
    await waitFor(() => expect(screen.getByText(/not switched on/)).toBeTruthy());
  });

  it("shows the store's prices exactly as given, and restores through the store", async () => {
    const restore = jest.fn(() => Promise.resolve());
    mockPrices = {
      plusYearly: "€24.99",
      pass: { weekend: "€2.99", week: "€4.99", long: "€6.99" },
      restore,
    };
    mockApi(entitlements("free"));
    await renderRouter(routes, { initialUrl: "/plans" });
    expect(await screen.findByText("€24.99")).toBeTruthy();
    expect(screen.getByLabelText("Week, 5 to 10 days: €4.99")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Restore purchases" }));
    await waitFor(() => expect(screen.getByText(/purchases are back/)).toBeTruthy());
    expect(restore).toHaveBeenCalledTimes(1);
  });

  it("opens the paywall when any call answers limit_reached", async () => {
    mockApi({
      "GET /v1/me/entitlements": () =>
        json(403, { code: "limit_reached", message: "Limit.", feature: "multiCurrency" }),
    });
    await renderRouter(routes, { initialUrl: "/plans" });
    expect(await screen.findByText("Add a second currency?")).toBeTruthy();
  });
});
