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

const flight = { kind: "flight", title: "VN 123", provider: "Vietnam Airlines" };
const stay = { kind: "stay", title: "Old Quarter hotel" };

const paste = async (text = "Your booking is confirmed") => {
  await renderRouter(routes, { initialUrl: "/trips/t1/import-booking" });
  await fireEvent.changeText(await screen.findByLabelText("Confirmation text"), text);
  await fireEvent.press(screen.getByRole("button", { name: "Find bookings" }));
};

describe("Import bookings", () => {
  it("shows what was found and adds only the ticked ones", async () => {
    const calls = mockApi({
      "POST /v1/trips/t1/bookings/import": () =>
        json(200, {
          candidates: [
            { booking: flight, confidence: 0.9 },
            { booking: stay, confidence: 0.3 },
          ],
        }),
      "POST /v1/trips/t1/bookings": () => json(201, { id: "b1", ...flight }),
    });
    await paste();
    expect(await screen.findByText("VN 123")).toBeTruthy();
    expect(screen.getAllByText("Check this one")).toHaveLength(1);
    await fireEvent.press(screen.getByRole("button", { name: "Include Old Quarter hotel" }));
    await fireEvent.press(screen.getByRole("button", { name: "Add 1 booking" }));
    await waitFor(() =>
      expect(
        calls.filter((c) => c.key === "POST /v1/trips/t1/bookings").map((c) => c.body),
      ).toEqual([flight]),
    );
    expect(calls.find((c) => c.key === "POST /v1/trips/t1/bookings/import")?.body).toEqual({
      text: "Your booking is confirmed",
    });
  });

  it("says so when nothing was found", async () => {
    mockApi({ "POST /v1/trips/t1/bookings/import": () => json(200, { candidates: [] }) });
    await paste();
    expect(await screen.findByText(/couldn't find a booking in that text/)).toBeTruthy();
  });

  it("explains when importing is switched off", async () => {
    mockApi({
      "POST /v1/trips/t1/bookings/import": () =>
        json(503, { code: "unavailable", message: "off", requestId: "r" }),
    });
    await paste();
    expect(await screen.findByText(/Importing isn't available/)).toBeTruthy();
  });

  it("asks for text first", async () => {
    mockApi({});
    await renderRouter(routes, { initialUrl: "/trips/t1/import-booking" });
    await fireEvent.press(await screen.findByRole("button", { name: "Find bookings" }));
    expect(await screen.findByText("Paste some text first.")).toBeTruthy();
  });
});
