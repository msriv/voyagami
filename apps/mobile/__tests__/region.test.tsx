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
const mockLocales = getLocales as jest.Mock;

beforeEach(() => {
  mockStore.clear();
  mockLocales.mockReturnValue([{ regionCode: "IN", languageTag: "en-IN", languageCode: "en" }]);
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
});

function api() {
  return mockApi({
    "GET /v1/me": () => json(200, { id: "u1", displayName: "Asha" }),
    "PATCH /v1/me": () => json(200, { id: "u1" }),
  });
}

describe("home region", () => {
  it("is the first onboarding step, filled from the phone, and saves what was shown", async () => {
    const calls = api();
    await renderRouter(routes, { initialUrl: "/" });
    expect(await screen.findByText("Where is home?")).toBeTruthy();
    expect(screen.getByText("India")).toBeTruthy();
    expect(screen.getByDisplayValue("INR")).toBeTruthy();

    await fireEvent.press(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText("How do you like to travel?")).toBeTruthy();
    const patch = calls.find((c) => c.key === "PATCH /v1/me");
    expect(patch?.body).toMatchObject({
      homeRegion: "IN",
      homeCurrency: "INR",
      regionSource: "confirmed",
      units: { distance: "km", temperature: "c" },
    });
  });

  it("lets the person pick another country, which resets currency and units", async () => {
    const calls = api();
    await renderRouter(routes, { initialUrl: "/" });
    await screen.findByText("Where is home?");
    await fireEvent.press(screen.getByRole("button", { name: "Change country" }));
    await fireEvent.changeText(await screen.findByLabelText("Search countries"), "united st");
    await fireEvent.press(screen.getByRole("button", { name: "United States" }));

    expect(screen.getByDisplayValue("USD")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Miles" }).props.accessibilityState.selected).toBe(
      true,
    );
    await fireEvent.press(screen.getByRole("button", { name: "Kilometres" }));
    await fireEvent.press(screen.getByRole("button", { name: "Continue" }));
    await screen.findByText("How do you like to travel?");
    expect(calls.find((c) => c.key === "PATCH /v1/me")?.body).toMatchObject({
      homeRegion: "US",
      homeCurrency: "USD",
      units: { distance: "km", temperature: "f" },
    });
  });

  it("needs a country when the phone does not say, and a valid currency code", async () => {
    mockLocales.mockReturnValue([{ regionCode: null, languageTag: "en", languageCode: "en" }]);
    api();
    await renderRouter(routes, { initialUrl: "/" });
    await screen.findByText("Where is home?");
    expect(screen.getByRole("button", { name: "Continue" })).toBeDisabled();

    await fireEvent.press(screen.getByRole("button", { name: "Choose country" }));
    await fireEvent.changeText(await screen.findByLabelText("Search countries"), "viet");
    await fireEvent.press(screen.getByRole("button", { name: "Vietnam" }));
    expect(screen.getByDisplayValue("VND")).toBeTruthy();

    await fireEvent.changeText(screen.getByLabelText("Currency"), "VN");
    expect(screen.getByRole("button", { name: "Continue" })).toBeDisabled();
    expect(screen.getByText("Use a three-letter currency code.")).toBeTruthy();
  });

  it("says so when nothing matches the search", async () => {
    api();
    await renderRouter(routes, { initialUrl: "/" });
    await screen.findByText("Where is home?");
    await fireEvent.press(screen.getByRole("button", { name: "Change country" }));
    await fireEvent.changeText(await screen.findByLabelText("Search countries"), "zzzzzz");
    expect(await screen.findByText("No country matches that.")).toBeTruthy();
  });
});
