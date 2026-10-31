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

const url = "/trips/t1/add-place?date=2099-11-19";
const created = {
  id: "n1",
  tripId: "t1",
  date: "2099-11-19",
  order: "a",
  place: { name: "Pho Thin" },
  timeZone: "Asia/Ho_Chi_Minh",
};

describe("Add a place by hand", () => {
  it("adds the place with its address, kind, pin and notes", async () => {
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "POST /v1/trips/t1/stops": () => json(201, created),
    });
    await renderRouter(routes, { initialUrl: url });
    await fireEvent.changeText(await screen.findByLabelText("Place name"), "Pho Thin");
    await fireEvent.changeText(screen.getByLabelText("Address (optional)"), "13 Lo Duc");
    await fireEvent.press(screen.getByRole("button", { name: "Food" }));
    await fireEvent.changeText(screen.getByLabelText("Latitude"), "21.0155");
    await fireEvent.changeText(screen.getByLabelText("Longitude"), "105.8571");
    await fireEvent.changeText(screen.getByLabelText("Notes (optional)"), "Cash only");
    await fireEvent.press(screen.getByRole("button", { name: "Add to itinerary" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/stops")?.body).toEqual({
        date: "2099-11-19",
        place: {
          name: "Pho Thin",
          lat: 21.0155,
          lng: 105.8571,
          address: "13 Lo Duc",
          category: "food",
        },
        note: "Cash only",
      }),
    );
  });

  it("needs a name and a sensible pin before calling the server", async () => {
    const calls = mockApi({ "GET /v1/me": () => json(200, { id: "u1" }) });
    await renderRouter(routes, { initialUrl: url });
    await fireEvent.press(await screen.findByRole("button", { name: "Add to itinerary" }));
    expect(await screen.findByText("Enter a place name.")).toBeTruthy();

    await fireEvent.changeText(screen.getByLabelText("Place name"), "Pho Thin");
    await fireEvent.changeText(screen.getByLabelText("Latitude"), "95");
    await fireEvent.press(screen.getByRole("button", { name: "Add to itinerary" }));
    expect(await screen.findByText(/Give both coordinates as numbers/)).toBeTruthy();
    expect(calls.some((c) => c.key.startsWith("POST"))).toBe(false);
  });

  it("tells the person when the server says no", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "POST /v1/trips/t1/stops": () => json(500, { code: "internal", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: url });
    await fireEvent.changeText(await screen.findByLabelText("Place name"), "Pho Thin");
    await fireEvent.press(screen.getByRole("button", { name: "Add to itinerary" }));
    expect(await screen.findByText("Something went wrong. Try again.")).toBeTruthy();
  });
});

describe("Search for a place", () => {
  it("fills the form from a result and saves its pin", async () => {
    const calls = mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/places/search": () =>
        json(200, {
          results: [
            { name: "Pho 10", address: "10 Ly Quoc Su", lat: 21.03, lng: 105.85, category: "food" },
          ],
        }),
      "POST /v1/trips/t1/stops": () => json(201, created),
    });
    await renderRouter(routes, { initialUrl: url });
    await fireEvent.changeText(await screen.findByLabelText("Search for a place"), "pho");
    await fireEvent.press(await screen.findByRole("button", { name: "Pho 10, 10 Ly Quoc Su" }));
    expect(screen.getByLabelText("Place name").props.value).toBe("Pho 10");
    await fireEvent.press(screen.getByRole("button", { name: "Add to itinerary" }));
    await waitFor(() =>
      expect(calls.find((c) => c.key === "POST /v1/trips/t1/stops")?.body).toMatchObject({
        place: {
          name: "Pho 10",
          lat: 21.03,
          lng: 105.85,
          address: "10 Ly Quoc Su",
          category: "food",
        },
      }),
    );
  });

  it("points to hand entry when search is down", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "GET /v1/places/search": () => json(503, { code: "unavailable", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: url });
    await fireEvent.changeText(await screen.findByLabelText("Search for a place"), "pho");
    expect(await screen.findByText(/Search isn't available/)).toBeTruthy();
  });
});

describe("Paste a link", () => {
  it("fills the form from the place in the link", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "POST /v1/places/resolve": () =>
        json(200, { name: "Hoan Kiem Lake", lat: 21.0285, lng: 105.8542 }),
    });
    await renderRouter(routes, { initialUrl: url });
    await fireEvent.changeText(
      await screen.findByLabelText("Or paste a map link"),
      "https://maps.app.goo.gl/x",
    );
    await fireEvent.press(screen.getByRole("button", { name: "Find place" }));
    await waitFor(() =>
      expect(screen.getByLabelText("Place name").props.value).toBe("Hoan Kiem Lake"),
    );
    expect(screen.getByLabelText("Latitude").props.value).toBe("21.0285");
  });

  it("says so when the link has no place", async () => {
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "POST /v1/places/resolve": () => json(404, { code: "not_found", message: "x" }),
    });
    await renderRouter(routes, { initialUrl: url });
    await fireEvent.changeText(
      await screen.findByLabelText("Or paste a map link"),
      "https://example.com",
    );
    await fireEvent.press(screen.getByRole("button", { name: "Find place" }));
    expect(await screen.findByText(/Couldn't find a place in that link/)).toBeTruthy();
  });
});
