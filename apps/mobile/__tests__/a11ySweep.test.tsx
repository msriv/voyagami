import { Pressable, Text } from "react-native";
import { render, renderRouter, screen } from "expo-router/testing-library";

import { inaccessibleControls } from "../test-utils/a11y";
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

const trip = {
  id: "t1",
  title: "Hanoi food week",
  ownerId: "u1",
  destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
  dates: { start: "2099-11-19", end: "2099-11-21" },
  timeZone: "Asia/Ho_Chi_Minh",
  settlementCurrency: "VND",
  status: "planned",
  role: "owner",
};

const api = {
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/me/entitlements": () => json(200, { plan: "free", features: {} }),
  "GET /v1/plans/free": () => json(200, { plan: "free", features: {} }),
  "GET /v1/me/stamps": () => json(200, { stamps: [], earned: 0, total: 5 }),
  "GET /v1/trips": () => json(200, { trips: [trip] }),
  "GET /v1/trips/t1": () => json(200, trip),
  "GET /v1/trips/t1/tasks": () => json(200, { tasks: [] }),
  "GET /v1/trips/t1/members": () => json(200, { members: [] }),
  "GET /v1/trips/t1/entitlements": () => json(200, { plan: "free", features: {} }),
  "GET /v1/trips/t1/recommendations": () =>
    json(200, { category: "eat", items: [], hasMore: false }),
};

const screens = [
  "/",
  "/profile",
  "/explore",
  "/plans",
  "/help",
  "/privacy",
  "/manage-subscription",
  "/trips/t1",
  "/trips/t1/picks",
  "/trips/t1/stays",
];

describe("Screen reader sweep", () => {
  it.each(screens)("%s has a role and a name on every control", async (url) => {
    mockApi(api);
    await renderRouter(routes, { initialUrl: url });
    await screen.findAllByRole("button");
    expect(inaccessibleControls()).toEqual([]);
  });
});

describe("Sweep helper", () => {
  it("flags a touchable with no role or name, and passes a proper button", async () => {
    await render(<Pressable onPress={() => undefined} />);
    expect(inaccessibleControls()).toHaveLength(1);
    await screen.unmount();
    await render(
      <Pressable accessibilityRole="button" onPress={() => undefined}>
        <Text>Go</Text>
      </Pressable>,
    );
    expect(inaccessibleControls()).toEqual([]);
  });
});
