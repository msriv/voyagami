import { getLocales } from "expo-localization";
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
// i18n reads the locale while it loads, so the mock must work before the tests run.
jest.mock("expo-localization", () => ({
  getLocales: jest.fn(() => [{ regionCode: "IN", languageTag: "en-IN", languageCode: "en" }]),
}));

beforeEach(() => {
  mockStore.clear();
  (getLocales as jest.Mock).mockClear();
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
  mockStore.set("voyagami.onboarded", "1");
});

const confirmed = {
  id: "u1",
  displayName: "Asha",
  homeRegion: "VN",
  homeCurrency: "VND",
  regionSource: "confirmed",
  units: { distance: "km", temperature: "c", clock: "24h", firstWeekday: 1 },
};

describe("home region defaults", () => {
  it("defaults a new trip's money from the saved home currency", async () => {
    mockApi({
      "GET /v1/me": () => json(200, confirmed),
      "GET /v1/trips": () => json(200, { trips: [] }),
    });
    await renderRouter(routes, { initialUrl: "/trips/new" });
    expect(await screen.findByDisplayValue("VND")).toBeTruthy();
  });

  it("ignores the placeholder region and currency saved at sign-up and uses the phone", async () => {
    mockApi({
      "GET /v1/me": () =>
        json(200, { id: "u1", homeRegion: "ZZ", homeCurrency: "XXX", regionSource: "device" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/new" });
    expect(await screen.findByDisplayValue("INR")).toBeTruthy();
  });

  it("switches region from Profile and shows the new one straight away", async () => {
    let me: Record<string, unknown> = { ...confirmed, homeRegion: "IN", homeCurrency: "INR" };
    const calls = mockApi({
      "GET /v1/me": () => json(200, me),
      "GET /v1/trips": () => json(200, { trips: [] }),
      "PATCH /v1/me": () => {
        me = confirmed;
        return json(200, me);
      },
    });
    await renderRouter(routes, { initialUrl: "/profile" });
    expect(await screen.findByText("India · INR")).toBeTruthy();

    await fireEvent.press(screen.getByRole("button", { name: "Change country" }));
    await screen.findByText("Where is home?");
    await fireEvent.press(screen.getByRole("button", { name: "Change country" }));
    await fireEvent.changeText(await screen.findByLabelText("Search countries"), "viet");
    await fireEvent.press(screen.getByRole("button", { name: "Vietnam" }));
    await fireEvent.press(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByText("Vietnam · VND")).toBeTruthy();
    expect(calls.find((c) => c.key === "PATCH /v1/me")?.body).toMatchObject({
      homeRegion: "VN",
      homeCurrency: "VND",
    });
  });

  it("shows who is signed in, their units and where their data lives", async () => {
    mockApi({
      "GET /v1/me": () => json(200, confirmed),
      "GET /v1/trips": () => json(200, { trips: [] }),
    });
    await renderRouter(routes, { initialUrl: "/profile" });
    expect(await screen.findByText("Asha")).toBeTruthy();
    expect(screen.getByText("Kilometres · Celsius · 24-hour clock")).toBeTruthy();
    expect(screen.getByText("Your data is stored in Mumbai, India")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Sign out" })).toBeTruthy();
  });
});
