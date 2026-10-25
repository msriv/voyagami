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

const stop = (id: string, name: string, date: string) => ({
  id,
  tripId: "t1",
  date,
  order: id,
  place: { name, lat: 21, lng: 105 },
  timeZone: "Asia/Ho_Chi_Minh",
});

const base = (role = "owner", entries: object[] = []) => ({
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1": () => json(200, trip(role)),
  "GET /v1/trips/t1/journal": () => json(200, { entries }),
  "GET /v1/trips/t1/stops": () =>
    json(200, {
      stops: [stop("s1", "Egg coffee", "2099-11-19"), stop("s2", "Ha Long", "2099-11-20")],
    }),
});

describe("Journal", () => {
  it("shows the day's plan and saves an entry with its stops", async () => {
    const calls = mockApi({
      ...base(),
      "PUT /v1/trips/t1/journal/2099-11-19": () =>
        json(200, {
          date: "2099-11-19",
          authorUid: "u1",
          visibility: "private",
          text: "Great coffee",
          mood: "happy",
          stopIds: ["s1"],
        }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/journal" });
    expect(await screen.findByText("Egg coffee")).toBeTruthy();
    expect(screen.queryByText("Ha Long")).toBeNull();
    await fireEvent.changeText(screen.getByLabelText("How was the day?"), "Great coffee");
    await fireEvent.press(screen.getByRole("button", { name: "Happy" }));
    await fireEvent.press(screen.getByRole("button", { name: "Just me" }));
    await fireEvent.press(screen.getByRole("button", { name: "Save entry" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PUT /v1/trips/t1/journal/2099-11-19")?.body).toEqual({
        text: "Great coffee",
        mood: "happy",
        visibility: "private",
        stopIds: ["s1"],
      }),
    );
    expect(await screen.findByText("Saved.")).toBeTruthy();
  });

  it("loads your entry, shows the crew's, and deletes yours", async () => {
    const calls = mockApi({
      ...base("owner", [
        { date: "2099-11-19", authorUid: "u1", visibility: "crew", text: "Mine", stopIds: [] },
        { date: "2099-11-19", authorUid: "u2", visibility: "crew", text: "Theirs", stopIds: [] },
      ]),
      "DELETE /v1/trips/t1/journal/2099-11-19": () => json(204),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/journal" });
    expect(await screen.findByDisplayValue("Mine")).toBeTruthy();
    expect(screen.getByText("Theirs")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Delete this entry" }));
    await waitFor(() =>
      expect(calls.some((c) => c.key === "DELETE /v1/trips/t1/journal/2099-11-19")).toBe(true),
    );
    await waitFor(() => expect(screen.queryByDisplayValue("Mine")).toBeNull());
  });

  it("asks for some text first", async () => {
    mockApi(base());
    await renderRouter(routes, { initialUrl: "/trips/t1/journal" });
    await fireEvent.press(await screen.findByRole("button", { name: "Save entry" }));
    expect(await screen.findByText("Write a line or two first.")).toBeTruthy();
  });

  it("is read-only for viewers", async () => {
    mockApi(
      base("viewer", [
        { date: "2099-11-19", authorUid: "u2", visibility: "crew", text: "Theirs", stopIds: [] },
      ]),
    );
    await renderRouter(routes, { initialUrl: "/trips/t1/journal" });
    expect(await screen.findByText("Theirs")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save entry" })).toBeNull();
  });
});
