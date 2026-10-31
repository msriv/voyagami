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
  mockApi({
    "GET /v1/me": () => json(200, { id: "u1" }),
    "GET /v1/trips": () => json(200, { trips: [] }),
  });
});

describe("Tour", () => {
  it("walks the steps, saves progress and can be skipped", async () => {
    await renderRouter(routes, { initialUrl: "/" });
    expect(await screen.findByText("Your trip lives here")).toBeTruthy();
    expect(screen.getByText("Step 1 of 5")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByText("Let the day sort itself")).toBeTruthy();
    expect(mockStore.get("voyagami.tour")).toBe("1");
    await fireEvent.press(screen.getByRole("button", { name: "Back" }));
    expect(await screen.findByText("Your trip lives here")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Skip tour" }));
    await waitFor(() => expect(screen.queryByText("Your trip lives here")).toBeNull());
    expect(mockStore.get("voyagami.tour")).toBe("done");
  });

  it("picks up where it stopped and stays away once finished", async () => {
    mockStore.set("voyagami.tour", "4");
    await renderRouter(routes, { initialUrl: "/" });
    expect(await screen.findByText("Your day writes itself")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Finish tour" }));
    await waitFor(() => expect(screen.queryByText("Your day writes itself")).toBeNull());
    expect(mockStore.get("voyagami.tour")).toBe("done");
  });

  it("stays away when it was already done", async () => {
    mockStore.set("voyagami.tour", "done");
    await renderRouter(routes, { initialUrl: "/" });
    await waitFor(() => expect(screen.queryByText(/Step \d of/)).toBeNull());
  });
});
