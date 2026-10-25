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

const comment = (id: string, authorUid: string, body: string) => ({
  id,
  targetType: "stop",
  targetId: "s1",
  authorUid,
  body,
  mentions: [],
  createdAt: "2099-11-19T08:00:00Z",
});

const base = (role = "owner", comments: object[] = []) => ({
  "GET /v1/me": () => json(200, { id: "u1" }),
  "GET /v1/trips/t1": () =>
    json(200, {
      id: "t1",
      title: "Hanoi food week",
      ownerId: "u1",
      destinations: [{ name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh" }],
      dates: {},
      timeZone: "Asia/Ho_Chi_Minh",
      settlementCurrency: "VND",
      status: "planned",
      role,
    }),
  "GET /v1/trips/t1/members": () =>
    json(200, {
      members: [
        { uid: "u1", role: "owner", status: "active", displayName: "Asha" },
        { uid: "u2", role: "editor", status: "active", displayName: "Ben" },
      ],
    }),
  "GET /v1/trips/t1/comments": () => json(200, { comments }),
});

describe("Comments", () => {
  it("posts a comment on a stop", async () => {
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/comments": () => json(201, comment("c1", "u1", "Book ahead")),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/comments?stopId=s1&title=Pho%2010" });
    await fireEvent.changeText(await screen.findByLabelText("Write a comment"), "Book ahead");
    await fireEvent.press(screen.getByRole("button", { name: "Send" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/comments")?.body).toEqual({
        targetType: "stop",
        targetId: "s1",
        body: "Book ahead",
      }),
    );
  });

  it("comments on the whole trip when there is no stop", async () => {
    const calls = mockApi({
      ...base(),
      "POST /v1/trips/t1/comments": () => json(201, comment("c1", "u1", "Hi")),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/comments" });
    await fireEvent.changeText(await screen.findByLabelText("Write a comment"), "Hi");
    await fireEvent.press(screen.getByRole("button", { name: "Send" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/comments")?.body).toEqual({
        targetType: "trip",
        targetId: "t1",
        body: "Hi",
      }),
    );
  });

  it("lets the author edit, and the owner delete someone else's", async () => {
    const calls = mockApi({
      ...base("owner", [comment("c1", "u1", "Mine"), comment("c2", "u2", "Theirs")]),
      "PATCH /v1/trips/t1/comments/c1": () => json(200, comment("c1", "u1", "Mine, edited")),
      "DELETE /v1/trips/t1/comments/c2": () => json(204, undefined),
    });
    await renderRouter(routes, { initialUrl: "/trips/t1/comments?stopId=s1" });
    await fireEvent.press(await screen.findByRole("button", { name: "You: Mine" }));
    await fireEvent.changeText(screen.getByLabelText("Edit comment"), "Mine, edited");
    await fireEvent.press(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "PATCH /v1/trips/t1/comments/c1")?.body).toEqual({
        body: "Mine, edited",
      }),
    );
    await fireEvent.press(await screen.findByRole("button", { name: "Ben: Theirs" }));
    expect(screen.queryByLabelText("Edit comment")).toBeNull();
    await fireEvent.press(screen.getByRole("button", { name: "Delete comment" }));
    await waitFor(() =>
      expect(calls.some((c) => c.key === "DELETE /v1/trips/t1/comments/c2")).toBe(true),
    );
  });
});
