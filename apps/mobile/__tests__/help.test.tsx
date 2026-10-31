import { Linking } from "react-native";
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
  delete process.env.EXPO_PUBLIC_SUPPORT_EMAIL;
  mockStore.clear();
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
  mockStore.set("voyagami.onboarded", "1");
  mockStore.set("voyagami.tour", "done");
});

const api = { "GET /v1/me": () => json(200, { id: "u1" }) };

describe("Help", () => {
  it("opens and closes an answer, announcing the state", async () => {
    mockApi(api);
    await renderRouter(routes, { initialUrl: "/help" });
    const q = await screen.findByRole("button", { name: "How do I cancel Plus?" });
    expect(q.props.accessibilityState).toMatchObject({ expanded: false });
    await fireEvent.press(q);
    expect(screen.getByText(/The store handles subscriptions/)).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "How do I cancel Plus?" }).props.accessibilityState,
    ).toMatchObject({ expanded: true });
  });

  it("says contact is not set up when there is no support address", async () => {
    mockApi(api);
    await renderRouter(routes, { initialUrl: "/help" });
    expect(await screen.findByText("Contact is not set up yet.")).toBeTruthy();
  });

  it("opens an email with diagnostics and nothing personal", async () => {
    process.env.EXPO_PUBLIC_SUPPORT_EMAIL = "help@example.com";
    const open = jest.spyOn(Linking, "openURL").mockResolvedValue();
    mockApi(api);
    await renderRouter(routes, { initialUrl: "/help" });
    await fireEvent.press(await screen.findByRole("button", { name: "Contact us" }));
    const url = decodeURIComponent(String(open.mock.calls[0]?.[0]));
    expect(url).toContain("mailto:help@example.com");
    expect(url).toMatch(/App .*\n.* .*\nLanguage /);
  });
});
