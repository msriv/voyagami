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
  mockStore.clear();
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
});

const me = { "GET /v1/me": () => json(200, { id: "u1" }) };

describe("new trip", () => {
  it("asks for a stop before saving", async () => {
    const calls = mockApi(me);
    await renderRouter(routes, { initialUrl: "/trips/new" });
    await fireEvent.press(await screen.findByRole("button", { name: "Create trip" }));
    expect(await screen.findByText("Add at least one stop.")).toBeTruthy();
    expect(calls.some((c) => c.key === "POST /v1/trips")).toBe(false);
  });

  it("asks for both days when dates are chosen", async () => {
    mockApi(me);
    await renderRouter(routes, { initialUrl: "/trips/new" });
    await fireEvent.press(await screen.findByRole("button", { name: "Hanoi" }));
    await fireEvent.press(screen.getByRole("button", { name: "Create trip" }));
    expect(await screen.findByText("Choose the first and last day.")).toBeTruthy();
  });

  it("saves a flexible trip through the API and returns Home", async () => {
    const calls = mockApi({
      ...me,
      "POST /v1/trips": () => json(201, { id: "t1" }),
    });
    await renderRouter(routes, { initialUrl: "/trips/new" });
    await fireEvent.press(await screen.findByRole("button", { name: "Hanoi" }));
    await fireEvent.press(screen.getByRole("button", { name: "Flexible" }));
    await fireEvent.press(screen.getByRole("button", { name: "One day more" }));
    await fireEvent.changeText(screen.getByLabelText("Settlement currency"), "inr");
    await fireEvent.press(screen.getByRole("button", { name: "Create trip" }));

    expect(await screen.findByText("Where to next?")).toBeTruthy();
    const post = calls.find((c) => c.key === "POST /v1/trips");
    expect(post?.body).toMatchObject({
      title: "Hanoi trip",
      timeZone: "Asia/Ho_Chi_Minh",
      settlementCurrency: "INR",
      dates: { flexible: { days: 8 } },
      destinations: [{ name: "Hanoi", countryCode: "VN" }],
    });
  });

  it("saves fixed dates picked on the calendar", async () => {
    const calls = mockApi({ ...me, "POST /v1/trips": () => json(201, { id: "t1" }) });
    await renderRouter(routes, { initialUrl: "/trips/new" });
    await fireEvent.press(await screen.findByRole("button", { name: "Tokyo" }));
    // The calendar opens on the current month; day buttons are labelled with their full date.
    const days = screen.getAllByRole("button", { name: /\b(1|2|3)\b.*\d{4}/ }).slice(0, 2);
    for (const day of days) await fireEvent.press(day);
    await fireEvent.changeText(screen.getByLabelText("Settlement currency"), "JPY");
    await fireEvent.press(screen.getByRole("button", { name: "Create trip" }));
    expect(await screen.findByText("Where to next?")).toBeTruthy();
    const body = calls.find((c) => c.key === "POST /v1/trips")?.body as {
      dates: { start: string; end: string };
    };
    expect(body.dates.start < body.dates.end).toBe(true);
  });
});
