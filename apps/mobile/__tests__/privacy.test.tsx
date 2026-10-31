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

const mockShare = jest.fn(() => Promise.resolve());
jest.mock("expo-sharing", () => ({ shareAsync: (...a: unknown[]) => mockShare(...(a as [])) }));
jest.mock("expo-file-system", () => ({
  Paths: { cache: "/cache" },
  File: class {
    exists = false;
    uri = "file:///cache/voyagami-data.zip";
    delete() {}
    create() {}
    write() {}
  },
}));

beforeEach(() => {
  mockShare.mockClear();
  mockStore.clear();
  mockStore.set("voyagami.accessToken", "a");
  mockStore.set("voyagami.refreshToken", "r");
  mockStore.set("voyagami.onboarded", "1");
  mockStore.set("voyagami.tour", "done");
});

describe("Data and privacy", () => {
  it("starts analytics off, and saves a change on the phone", async () => {
    mockApi({ "GET /v1/me": () => json(200, { id: "u1" }) });
    await renderRouter(routes, { initialUrl: "/privacy" });
    const analytics = await screen.findByRole("switch", { name: "Help improve Voyagami" });
    expect(analytics.props.accessibilityState).toMatchObject({ checked: false });
    expect(
      screen.getByRole("switch", { name: "Share my location with my crew" }).props
        .accessibilityState,
    ).toMatchObject({ checked: true });
    await fireEvent(analytics, "valueChange", true);
    await waitFor(() =>
      expect(JSON.parse(mockStore.get("voyagami.consents") ?? "{}")).toMatchObject({
        analytics: true,
      }),
    );
  });

  it("remembers earlier choices", async () => {
    mockStore.set("voyagami.consents", JSON.stringify({ location: false }));
    mockApi({ "GET /v1/me": () => json(200, { id: "u1" }) });
    await renderRouter(routes, { initialUrl: "/privacy" });
    await waitFor(() =>
      expect(
        screen.getByRole("switch", { name: "Share my location with my crew" }).props
          .accessibilityState,
      ).toMatchObject({ checked: false }),
    );
  });

  it("sends a consent change to the server", async () => {
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "PATCH /v1/me/consents": () =>
        json(200, {
          location: "granted",
          notifications: "granted",
          photos: "granted",
          analytics: "granted",
        }),
    });
    await renderRouter(routes, { initialUrl: "/privacy" });
    await fireEvent(
      await screen.findByRole("switch", { name: "Help improve Voyagami" }),
      "valueChange",
      true,
    );
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/me/consents")?.body).toEqual({
        analytics: "granted",
      }),
    );
  });

  it("exports the data and opens the share sheet", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/me/export": () =>
        new Response(new Uint8Array([80, 75]), {
          status: 200,
          headers: { "content-type": "application/zip" },
        }),
    });
    await renderRouter(routes, { initialUrl: "/privacy" });
    await fireEvent.press(await screen.findByRole("button", { name: "Export my data" }));
    await waitFor(() => expect(mockShare).toHaveBeenCalled());
  });

  it("asks before scheduling deletion, then lets the person keep the account", async () => {
    let scheduled = false;
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/me/deletion": () => json(200, { scheduled }),
      "DELETE /v1/me": () => {
        scheduled = true;
        return json(200, { scheduled: true, deleteOn: "2026-12-01T00:00:00Z" });
      },
      "DELETE /v1/me/deletion": () => new Response(null, { status: 204 }),
    });
    await renderRouter(routes, { initialUrl: "/privacy" });
    await fireEvent.press(await screen.findByRole("button", { name: "Delete my account" }));
    expect(calls.some((c) => c.key === "DELETE /v1/me")).toBe(false);
    await fireEvent.press(await screen.findByRole("button", { name: "Yes, delete my account" }));
    expect(await screen.findByText(/Your account will be deleted on/)).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Keep my account" }));
    await waitFor(() => expect(calls.some((c) => c.key === "DELETE /v1/me/deletion")).toBe(true));
    expect(await screen.findByRole("button", { name: "Delete my account" })).toBeTruthy();
  });

  it("tells the person to cancel the subscription first", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/me/deletion": () => json(200, { scheduled: false }),
      "DELETE /v1/me": () => json(409, { code: "subscription_active", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/privacy" });
    await fireEvent.press(await screen.findByRole("button", { name: "Delete my account" }));
    await fireEvent.press(await screen.findByRole("button", { name: "Yes, delete my account" }));
    expect(await screen.findByText(/Cancel it in your store/)).toBeTruthy();
  });
});
