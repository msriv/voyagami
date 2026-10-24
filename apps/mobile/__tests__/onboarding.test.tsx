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

beforeEach(() => mockStore.clear());

describe("welcome", () => {
  it("opens first when signed out and leads to sign-in", async () => {
    mockApi({});
    await renderRouter(routes, { initialUrl: "/" });
    expect(await screen.findByText("Trips are better planned together.")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Get started" }));
    expect(await screen.findByLabelText("Mobile number")).toBeTruthy();
  });
});

describe("travel style", () => {
  async function signedInAtStyle() {
    mockStore.set("voyagami.accessToken", "a");
    mockStore.set("voyagami.refreshToken", "r");
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1", displayName: "Asha" }),
      "PATCH /v1/me": () => json(200, { id: "u1" }),
    });
    await renderRouter(routes, { initialUrl: "/onboarding/style" });
    await screen.findByText("How do you like to travel?");
    return calls;
  }

  it("saves picks, announces selection state and goes Home", async () => {
    const calls = await signedInAtStyle();
    await fireEvent.press(screen.getByRole("button", { name: "Beaches" }));
    await fireEvent.press(screen.getByRole("button", { name: "Slow cafes" }));
    await fireEvent.press(screen.getByRole("button", { name: "Slow" }));
    await fireEvent.press(screen.getByRole("button", { name: "Comfort" }));
    expect(screen.getByRole("button", { name: "Beaches" }).props.accessibilityState.selected).toBe(
      true,
    );
    await fireEvent.press(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText("Welcome, Asha")).toBeTruthy();
    const patch = calls.find((c) => c.key === "PATCH /v1/me");
    expect(patch?.body).toEqual({
      travelStyle: { interests: ["beaches", "slowCafes"], pace: "slow", spend: "comfort" },
    });
  });

  it("skips without calling the server", async () => {
    const calls = await signedInAtStyle();
    await fireEvent.press(screen.getByRole("button", { name: "Skip for now" }));
    expect(await screen.findByText("Welcome, Asha")).toBeTruthy();
    await waitFor(() => expect(calls.some((c) => c.key === "PATCH /v1/me")).toBe(false));
  });
});
