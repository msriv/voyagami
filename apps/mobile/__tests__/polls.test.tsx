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
  dates: {},
  timeZone: "Asia/Ho_Chi_Minh",
  settlementCurrency: "VND",
  status: "planned",
  role,
});

const poll = (over: object = {}) => ({
  id: "p1",
  question: "Dinner spot?",
  options: [
    { id: "a", label: "Bun cha", votes: 1 },
    { id: "b", label: "Pho", votes: 1 },
  ],
  multi: false,
  status: "open",
  createdBy: "u1",
  myVote: [],
  voters: 2,
  ...over,
});

const base = (role = "owner", polls: object[] = [poll()]) => ({
  "GET /v1/trips/t1": () => json(200, trip(role)),
  "GET /v1/trips/t1/polls": () => json(200, { polls }),
});

describe("Polls", () => {
  it("votes for an option", async () => {
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/polls/p1/vote": () => json(200, poll({ myVote: ["b"] })),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/polls" });
    await fireEvent.press(await screen.findByRole("button", { name: "Pho: 1" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/polls/p1/vote")?.body).toEqual({
        optionIds: ["b"],
      }),
    );
  });

  it("asks for a winner when closing a tie, then closes with it", async () => {
    let attempt = 0;
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/polls/p1/close": () =>
        attempt++ === 0
          ? json(409, { code: "conflict", message: "tie" })
          : json(200, {
              poll: poll({ status: "closed", result: { optionId: "a", decidedBy: "u1" } }),
            }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/polls" });
    await fireEvent.press(await screen.findByRole("button", { name: "Close poll" }));
    expect(await screen.findByText(/pick the winner/)).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Bun cha: 1" }));
    expect(await screen.findByText("Winner, Bun cha: 1")).toBeTruthy();
    expect(calls.filter((c) => c.key === "POST /v1/trips/t1/polls/p1/close").at(-1)?.body).toEqual({
      optionId: "a",
    });
  });

  it("hides start and close from viewers", async () => {
    mockApi(base("viewer"));
    await renderRouter(routes, { initialUrl: "/trips/t1/polls" });
    await screen.findByText("Dinner spot?");
    expect(screen.queryByRole("button", { name: "Start a poll" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Close poll" })).toBeNull();
  });

  it("starts a poll", async () => {
    const calls = mockApi({
      ...base("owner", []),
      "POST /v1/trips/t1/polls": () => json(201, poll()),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/polls" });
    await fireEvent.press(await screen.findByRole("button", { name: "Start a poll" }));
    await fireEvent.changeText(screen.getByLabelText("Question"), "Dinner spot?");
    await fireEvent.changeText(screen.getByLabelText("Option 1"), "Bun cha");
    await fireEvent.changeText(screen.getByLabelText("Option 2"), "Pho");
    await fireEvent.press(screen.getByRole("button", { name: "Start poll" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/polls")?.body).toEqual({
        question: "Dinner spot?",
        options: [{ label: "Bun cha" }, { label: "Pho" }],
        multi: false,
      }),
    );
  });
});
