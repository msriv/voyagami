import { Share } from "react-native";
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

const member = (uid: string, displayName: string, role: string, status = "active") => ({
  uid,
  displayName,
  role,
  status,
  joinedAt: "2099-01-01T00:00:00Z",
});

const base = (role = "owner", me = "u1") => ({
  "GET /v1/me": () => json(200, { id: me }),
  "GET /v1/trips/t1": () => json(200, trip(role)),
  "GET /v1/trips/t1/members": () =>
    json(200, {
      members: [
        member("u1", "Asha", "owner"),
        member("u2", "Ben", "editor"),
        member("u3", "Old friend", "editor", "left"),
      ],
    }),
  "GET /v1/trips/t1/invites": () =>
    json(200, {
      invites: [
        {
          id: "i1",
          role: "viewer",
          createdBy: "u1",
          createdAt: "2099-01-01T00:00:00Z",
          useCount: 2,
          revoked: false,
        },
      ],
    }),
});

describe("Crew", () => {
  it("lists active members and marks you", async () => {
    mockApi(base());
    await renderRouter(routes, { initialUrl: "/trips/t1/crew" });
    expect(await screen.findByText("Asha (you)")).toBeTruthy();
    expect(screen.getByText("Ben")).toBeTruthy();
    expect(screen.queryByText("Old friend")).toBeNull();
  });

  it("says a Trip Pass covers everyone", async () => {
    mockApi({
      ...base(),
      "GET /v1/trips/t1/entitlements": () =>
        json(200, { plan: "pass", features: {}, pass: { boughtBy: "u1" } }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/crew" });
    expect(await screen.findByText(/This trip has a Trip Pass/)).toBeTruthy();
  });

  it("shares a new invite link with the code, once", async () => {
    const share = jest
      .spyOn(Share, "share")
      .mockResolvedValue({ action: "sharedAction", activityType: undefined });
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/invites": () =>
        json(201, {
          invite: {
            id: "i2",
            role: "viewer",
            createdBy: "u1",
            createdAt: "2099-01-01T00:00:00Z",
            useCount: 0,
            revoked: false,
          },
          token: "secret-token",
        }),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/crew" });
    await fireEvent.press(await screen.findByRole("button", { name: "Can only view" }));
    await fireEvent.press(screen.getByRole("button", { name: "Share an invite link" }));
    await waitFor(() => expect(share).toHaveBeenCalled());
    expect(calls.find((c) => c.key === "POST /v1/trips/t1/invites")?.body).toEqual({
      role: "viewer",
    });
    expect(JSON.stringify(share.mock.calls[0])).toContain("secret-token");
  });

  it("revokes an invite and changes a member's role", async () => {
    const calls = mockApi({
      ...base(),
      "DELETE /v1/trips/t1/invites/i1": () => json(204),
      "PATCH /v1/trips/t1/members/u2": () => json(200, member("u2", "Ben", "viewer")),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/crew" });
    await fireEvent.press(
      await screen.findByRole("button", { name: "Stop the invite link for Viewer" }),
    );
    await waitFor(() =>
      expect(calls.some((c) => c.key === "DELETE /v1/trips/t1/invites/i1")).toBe(true),
    );
    await fireEvent.press(await screen.findByRole("button", { name: "Manage Ben" }));
    await fireEvent.press(screen.getByRole("button", { name: "Viewer" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/trips/t1/members/u2")?.body).toEqual({
        role: "viewer",
      }),
    );
  });

  it("lets a non-owner leave but not invite", async () => {
    const calls = mockApi({
      ...base("editor", "u2"),
      "DELETE /v1/trips/t1/members/u2": () => json(204),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/crew" });
    expect(await screen.findByText("Ben (you)")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Share an invite link" })).toBeNull();
    await fireEvent.press(screen.getByRole("button", { name: "Manage Ben" }));
    await fireEvent.press(screen.getByRole("button", { name: "Leave this trip" }));
    await waitFor(() =>
      expect(calls.some((c) => c.key === "DELETE /v1/trips/t1/members/u2")).toBe(true),
    );
  });
});

describe("Join", () => {
  it("joins with the code from the link and opens the trip", async () => {
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u2" }),
      "POST /v1/invites/join": () => json(200, trip("editor")),
      "GET /v1/trips/t1": () => json(200, trip("editor")),
      "GET /v1/trips/t1/tasks": () => json(200, { tasks: [] }),
    });
    const view = await renderRouter(routes, { initialUrl: "/join?token=abc123" });
    expect(await screen.findByDisplayValue("abc123")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Join trip" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/invites/join")?.body).toEqual({
        token: "abc123",
      }),
    );
    await waitFor(() => expect(view.getPathname()).toBe("/trips/t1"));
  });

  it("says when the invite is not valid", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u2" }),
      "POST /v1/invites/join": () => json(404, { code: "not_found", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: "/join" });
    await fireEvent.changeText(await screen.findByLabelText("Invite code"), "nope");
    await fireEvent.press(screen.getByRole("button", { name: "Join trip" }));
    expect(await screen.findByText(/isn't valid any more/)).toBeTruthy();
  });
});
