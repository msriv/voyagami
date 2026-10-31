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
  mockStore.set("voyagami.tour", "done");
});

describe("Data and privacy", () => {
  it("starts analytics off, and saves a change on the phone", async () => {
    mockApi({ "GET /v1/me": () => json(200, { id: "u1" }) });
    await renderRouter(routes, { initialUrl: "/privacy" });
    const analytics = await screen.findByRole("switch", { name: "Help improve Voyagami" });
    expect(analytics.props.accessibilityState).toMatchObject({ checked: false });
    expect(
      screen.getByRole("switch", { name: "Share my location with my crew" }).props
        .accessibilityState,
    ).toMatchObject({ checked: true });
    await fireEvent(analytics, "valueChange", true);
    await waitFor(() =>
      expect(JSON.parse(mockStore.get("voyagami.consents") ?? "{}")).toMatchObject({
        analytics: true,
      }),
    );
  });

  it("remembers earlier choices", async () => {
    mockStore.set("voyagami.consents", JSON.stringify({ location: false }));
    mockApi({ "GET /v1/me": () => json(200, { id: "u1" }) });
    await renderRouter(routes, { initialUrl: "/privacy" });
    await waitFor(() =>
      expect(
        screen.getByRole("switch", { name: "Share my location with my crew" }).props
          .accessibilityState,
      ).toMatchObject({ checked: false }),
    );
  });
});
