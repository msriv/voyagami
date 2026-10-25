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
});

const api = (transfers: object[]) => ({
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1/members": () =>
    json(200, {
      members: [
        { uid: "u1", role: "owner", status: "active", displayName: "Asha" },
        { uid: "u2", role: "editor", status: "active", displayName: "Ben" },
      ],
    }),
  "GET /v1/trips/t1/balances": () =>
    json(200, {
      currency: "VND",
      balances: [
        { uid: "u1", amountMinor: 50000 },
        { uid: "u2", amountMinor: -50000 },
      ],
      transfers,
    }),
  "GET /v1/trips/t1/settlements": () => json(200, { settlements: [] }),
});

describe("Settle up", () => {
  it("shows who owes whom and records a payment", async () => {
    const calls = mockApi({
      ...api([{ fromUid: "u2", toUid: "u1", amountMinor: 50000 }]),
      "POST /v1/trips/t1/settlements": () => json(201, { id: "s1" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/settle" });
    expect(await screen.findByText(/Ben pays You/)).toBeTruthy();
    expect(screen.getByText(/Owes/)).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Mark as paid" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/settlements")?.body).toEqual({
        fromUid: "u2",
        toUid: "u1",
        amountMinor: 50000,
        method: "manual",
      }),
    );
  });

  it("says when everyone is square", async () => {
    mockApi(api([]));
    await renderRouter(routes, { initialUrl: "/trips/t1/settle" });
    expect(await screen.findByText("Everyone is square.")).toBeTruthy();
  });
});
