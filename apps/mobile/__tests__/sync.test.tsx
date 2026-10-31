import { fireEvent, renderRouter, screen, waitFor } from "expo-router/testing-library";

import { enqueue, failedWrites, flush, pendingWrites } from "../src/local/queue";
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

const expense = (id: string, tripId = "t1") => ({
  op: "createExpense" as const,
  tripId,
  id,
  body: {
    id,
    description: `Pho ${id}`,
    paidByUid: "u1",
    date: "2026-11-01",
    original: { amountMinor: 50000, currency: "VND" },
    split: { method: "equal" as const, parts: [{ uid: "u1" }] },
  },
});

describe("Changes to send", () => {
  it("says when nothing is waiting", async () => {
    mockApi({});
    await renderRouter(routes, { initialUrl: "/trips/t1/sync" });
    expect(await screen.findByText(/Nothing is waiting/)).toBeTruthy();
  });

  it("lists this trip's waiting changes and lets the person discard one", async () => {
    mockApi({});
    await enqueue(expense("a"));
    await enqueue(expense("other", "t2"));
    await renderRouter(routes, { initialUrl: "/trips/t1/sync" });
    expect(await screen.findByText(/Expense: Pho a, /)).toBeTruthy();
    expect(screen.queryByText(/Pho other/)).toBeNull();
    await fireEvent.press(screen.getByRole("button", { name: /^Discard: Expense: Pho a/ }));
    await waitFor(() => expect(screen.queryByText(/Pho a,/)).toBeNull());
    expect((await pendingWrites()).map((e) => e.id)).toEqual(["other"]);
  });

  it("explains a refused change and sends it again on request", async () => {
    const calls = mockApi({
      "POST /v1/trips/t1/expenses": () => json(201, { id: "a" }),
    });
    await enqueue(expense("a"));
    await flush(async () => ({ reject: "That day is not in the trip." }));
    expect(await failedWrites()).toHaveLength(1);
    await renderRouter(routes, { initialUrl: "/trips/t1/sync" });
    expect(await screen.findByText("Why: That day is not in the trip.")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: /^Try again: Expense: Pho a/ }));
    await waitFor(() =>
      expect(calls.some((c) => c.key === "POST /v1/trips/t1/expenses")).toBe(true),
    );
    await waitFor(() => expect(screen.queryByText(/^Why:/)).toBeNull());
    expect(await failedWrites()).toHaveLength(0);
  });
});
