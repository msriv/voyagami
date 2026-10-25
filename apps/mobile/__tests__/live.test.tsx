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
const mockPermission = jest.fn();
jest.mock("expo-location", () => ({
  Accuracy: { Balanced: 3 },
  requestForegroundPermissionsAsync: () => mockPermission(),
  getCurrentPositionAsync: jest.fn(() =>
    Promise.resolve({ coords: { latitude: 21.03, longitude: 105.85, accuracy: 12.4 } }),
  ),
  watchPositionAsync: jest.fn(() => Promise.resolve({ remove: jest.fn() })),
}));

beforeEach(() => {
  mockStore.clear();
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
  mockStore.set("voyagami.onboarded", "1");
  mockPermission.mockResolvedValue({ granted: true });
});

const soon = () => new Date(Date.now() + 3_600_000).toISOString();

const meet = (over: object = {}) => ({
  id: "m1",
  name: "Hoan Kiem lake",
  place: { name: "Hoan Kiem lake" },
  createdBy: "u1",
  confirmedBy: ["u2"],
  iAmComing: false,
  endsAt: soon(),
  status: "active",
  ...over,
});

const base = (locations: object[] = [], meets: object[] = []) => ({
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1": () =>
    json(200, {
      id: "t1",
      title: "Hanoi food week",
      ownerId: "u1",
      destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
      dates: {},
      timeZone: "Asia/Ho_Chi_Minh",
      settlementCurrency: "VND",
      status: "planned",
      role: "owner",
    }),
  "GET /v1/trips/t1/members": () =>
    json(200, {
      members: [
        { uid: "u1", role: "owner", status: "active", displayName: "Asha" },
        { uid: "u2", role: "editor", status: "active", displayName: "Ben" },
      ],
    }),
  "GET /v1/trips/t1/locations": () => json(200, { locations }),
  "GET /v1/trips/t1/meet-points": () => json(200, { meetPoints: meets }),
});

describe("Live", () => {
  it("shares for an hour after permission, rounding accuracy", async () => {
    const calls = mockApi({
      ...base(),
      "PUT /v1/trips/t1/locations/me": () => json(200, {}),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/live" });
    await fireEvent.press(await screen.findByRole("button", { name: "Share for 1 hour" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PUT /v1/trips/t1/locations/me")?.body).toEqual({
        lat: 21.03,
        lng: 105.85,
        accuracyM: 12,
        durationMinutes: 60,
      }),
    );
  });

  it("does not send a location when permission is refused", async () => {
    mockPermission.mockResolvedValue({ granted: false });
    const calls = mockApi(base());
    await renderRouter(routes, { initialUrl: "/trips/t1/live" });
    await fireEvent.press(await screen.findByRole("button", { name: "Share for 1 hour" }));
    expect(await screen.findByText(/Allow location in your phone settings/)).toBeTruthy();
    expect(calls.some((c) => c.key.startsWith("PUT"))).toBe(false);
  });

  it("shows a stop button while sharing, and who else is sharing", async () => {
    const calls = mockApi({
      ...base([
        { uid: "u1", lat: 1, lng: 2, accuracyM: 5, updatedAt: soon(), endsAt: soon() },
        { uid: "u2", lat: 1, lng: 2, accuracyM: 5, updatedAt: soon(), endsAt: soon() },
      ]),
      "DELETE /v1/trips/t1/locations/me": () => json(204, undefined),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/live" });
    expect(await screen.findByText("Ben")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Stop sharing" }));
    await waitFor(() =>
      expect(calls.some((c) => c.key === "DELETE /v1/trips/t1/locations/me")).toBe(true),
    );
  });

  it("says you are coming to a meet point", async () => {
    const calls = mockApi({
      ...base([], [meet()]),
      "PUT /v1/trips/t1/meet-points/m1/confirmation": () => json(204, undefined),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/live" });
    expect(await screen.findByText("1 coming")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "I'm coming" }));
    await waitFor(() =>
      expect(calls.some((c) => c.key === "PUT /v1/trips/t1/meet-points/m1/confirmation")).toBe(
        true,
      ),
    );
  });

  it("proposes a meet point", async () => {
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/meet-points": () => json(201, meet()),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/live" });
    await fireEvent.press(await screen.findByRole("button", { name: "Propose a meet point" }));
    await fireEvent.changeText(
      screen.getByLabelText("Where do you want to meet?"),
      "Hoan Kiem lake",
    );
    await fireEvent.press(screen.getByRole("button", { name: "Add meet point" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/meet-points")?.body).toEqual({
        name: "Hoan Kiem lake",
        place: { name: "Hoan Kiem lake" },
      }),
    );
  });
});
