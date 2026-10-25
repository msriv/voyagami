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

const trip = (role: string) => ({
  id: "t1",
  title: "Hanoi food week",
  ownerId: "u1",
  destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
  dates: { start: "2099-11-19", end: "2099-11-21" },
  timeZone: "Asia/Ho_Chi_Minh",
  settlementCurrency: "VND",
  status: "planned",
  role,
});

const expense = (
  id: string,
  description: string,
  date: string,
  amountMinor: number,
  by = "u1",
) => ({
  id,
  description,
  paidByUid: by,
  date,
  original: { amountMinor, currency: "VND" },
  settlement: { amountMinor, currency: "VND" },
  split: { method: "equal", shares: [{ uid: "u1", amountMinor }] },
  createdBy: "u1",
});

const spent = (expenses: object[]) =>
  (expenses as { settlement: { amountMinor: number } }[]).reduce(
    (n, x) => n + x.settlement.amountMinor,
    0,
  );

const base = (role = "owner", expenses: object[] = [], budget: object = {}) => ({
  "GET /v1/trips/t1/budget": () =>
    json(200, {
      currency: "VND",
      spentMinor: spent(expenses),
      overBudget: false,
      byCategory: [],
      ...budget,
    }),
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1": () => json(200, trip(role)),
  "GET /v1/trips/t1/expenses": () => json(200, { expenses }),
});

describe("Money", () => {
  it("lists spending by day with a total, in the trip's currency", async () => {
    mockApi(
      base("owner", [
        expense("e1", "Pho", "2099-11-20", 50000),
        expense("e2", "Taxi", "2099-11-19", 120000, "u2"),
      ]),
    );
    await renderRouter(routes, { initialUrl: "/trips/t1/money" });
    expect(await screen.findByText("Pho")).toBeTruthy();
    expect(screen.getByText("Taxi")).toBeTruthy();
    expect(screen.getByText("Paid by you")).toBeTruthy();
    expect(screen.getByText("Paid by someone else")).toBeTruthy();
    expect(screen.getByText(/170,000/)).toBeTruthy();
  });

  it("shows the empty state and hides Add for viewers", async () => {
    mockApi(base("viewer"));
    await renderRouter(routes, { initialUrl: "/trips/t1/money" });
    expect(await screen.findByText("No spending yet.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Add an expense" })).toBeNull();
  });

  it("deletes an expense", async () => {
    const calls = mockApi({
      ...base("owner", [expense("e1", "Pho", "2099-11-20", 50000)]),
      "DELETE /v1/trips/t1/expenses/e1": () => json(204),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/money" });
    await fireEvent.press(await screen.findByRole("button", { name: /^Pho/ }));
    await fireEvent.press(screen.getByRole("button", { name: "Delete this expense" }));
    await waitFor(() =>
      expect(calls.some((c) => c.key === "DELETE /v1/trips/t1/expenses/e1")).toBe(true),
    );
    await waitFor(() => expect(screen.queryByText("Pho")).toBeNull());
  });
});

describe("Budget", () => {
  it("shows what is left, or what is over", async () => {
    mockApi(
      base("owner", [expense("e1", "Pho", "2099-11-20", 50000)], {
        targetMinor: 200000,
        remainingMinor: 150000,
        byCategory: [{ category: "food", spentMinor: 50000 }],
      }),
    );
    await renderRouter(routes, { initialUrl: "/trips/t1/money" });
    expect(await screen.findByText(/150,000.*left of.*200,000/)).toBeTruthy();
    expect(screen.getByText("Food")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Change the budget" })).toBeTruthy();
  });

  it("lets the owner set a budget, and nobody else", async () => {
    const calls = mockApi({
      ...base("owner"),
      "PATCH /v1/trips/t1": () => json(200, trip("owner")),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/money" });
    await fireEvent.press(await screen.findByRole("button", { name: "Set a budget" }));
    await fireEvent.changeText(screen.getByLabelText(/^Budget in VND/), "2000000");
    await fireEvent.press(screen.getByRole("button", { name: "For each person" }));
    await fireEvent.press(screen.getByRole("button", { name: "Save budget" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/trips/t1")?.body).toEqual({
        budget: { scope: "perPerson", amount: { amountMinor: 2000000, currency: "VND" } },
      }),
    );
  });

  it("hides the budget button from editors", async () => {
    mockApi(base("editor"));
    await renderRouter(routes, { initialUrl: "/trips/t1/money" });
    expect(await screen.findByText("No spending yet.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Set a budget" })).toBeNull();
  });
});

describe("Add expense", () => {
  it("saves an equal split to yourself in minor units", async () => {
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/expenses": () => json(201, expense("e1", "Pho", "2099-11-19", 50000)),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/add-expense" });
    await fireEvent.changeText(await screen.findByLabelText("What was it for?"), "Pho");
    await fireEvent.changeText(screen.getByLabelText(/^Amount in VND/), "50000");
    await fireEvent.press(screen.getByRole("button", { name: "Save expense" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/expenses")?.body).toEqual({
        description: "Pho",
        paidByUid: "u1",
        date: "2099-11-19",
        original: { amountMinor: 50000, currency: "VND" },
        split: { method: "equal", parts: [{ uid: "u1" }] },
      }),
    );
  });

  it("asks for a valid amount and a description", async () => {
    mockApi(base());
    await renderRouter(routes, { initialUrl: "/trips/t1/add-expense" });
    await fireEvent.press(await screen.findByRole("button", { name: "Save expense" }));
    expect(await screen.findByText("Say what it was for.")).toBeTruthy();
    await fireEvent.changeText(screen.getByLabelText("What was it for?"), "Pho");
    await fireEvent.changeText(screen.getByLabelText(/^Amount in VND/), "12.5");
    await fireEvent.press(screen.getByRole("button", { name: "Save expense" }));
    expect(await screen.findByText(/Enter an amount more than zero/)).toBeTruthy();
  });
});
