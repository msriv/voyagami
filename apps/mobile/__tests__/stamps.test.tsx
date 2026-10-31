import { renderRouter, screen, waitFor } from "expo-router/testing-library";

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

const kinds = ["createTrip", "inviteCrew", "addPlaces", "logExpense", "writeJournal"];
const stamps = (earned: number) => ({
  stamps: kinds.map((kind, i) => ({ kind, earned: i < earned })),
  earned,
  total: 5,
});

describe("Stamp board", () => {
  it("shows the five stamps with what is earned and what is not", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips": () => json(200, { trips: [] }),
      "GET /v1/me/stamps": () => json(200, stamps(2)),
    });
    await renderRouter(routes, { initialUrl: "/" });
    expect(await screen.findByText("Collect your first five stamps")).toBeTruthy();
    expect(screen.getByLabelText("Create a trip, stamped")).toBeTruthy();
    expect(
      screen.getByLabelText("Add 3 places, not yet. Search, or paste a reel link"),
    ).toBeTruthy();
    expect(screen.getByText("2 of 5")).toBeTruthy();
  });

  it("leaves Home once all five are earned", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/trips": () => json(200, { trips: [] }),
      "GET /v1/me/stamps": () => json(200, stamps(5)),
    });
    await renderRouter(routes, { initialUrl: "/" });
    await waitFor(() => expect(screen.getByText("Your passport is empty. For now.")).toBeTruthy());
    expect(screen.queryByText("Collect your first five stamps")).toBeNull();
  });
});
