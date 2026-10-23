import { fireEvent, renderRouter, screen, waitFor } from "expo-router/testing-library";

import { json, mockApi, routes, tokens } from "../test-utils/api";

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

beforeEach(() => mockStore.clear());

async function enterNumber() {
  await renderRouter(routes, { initialUrl: "/sign-in" });
  await fireEvent.changeText(await screen.findByLabelText("Mobile number"), "+14155550123");
  await fireEvent.press(screen.getByRole("button", { name: "Send 6-digit code" }));
}

describe("sign in", () => {
  it("sends a code, verifies it, keeps the session and shows Home with the name", async () => {
    const calls = mockApi({
      "POST /v1/auth/otp/request": () => json(204),
      "POST /v1/auth/otp/verify": () => json(200, tokens),
      "GET /v1/me": () => json(200, { id: "u1", displayName: "Asha" }),
    });
    await enterNumber();
    await fireEvent.changeText(await screen.findByLabelText("6-digit code"), "123456");
    await fireEvent.press(screen.getByRole("button", { name: "Verify" }));

    expect(await screen.findByText("Welcome, Asha")).toBeTruthy();
    expect(calls[0]).toEqual({ key: "POST /v1/auth/otp/request", body: { phone: "+14155550123" } });
    expect(calls[1]?.body).toMatchObject({ phone: "+14155550123", code: "123456" });
    expect(mockStore.get("voyagami.refreshToken")).toBe("refresh-1");
  });

  it("explains a wrong code and stays on the code screen", async () => {
    mockApi({
      "POST /v1/auth/otp/request": () => json(204),
      "POST /v1/auth/otp/verify": () => json(400, { code: "invalid_code", message: "Wrong code." }),
    });
    await enterNumber();
    await fireEvent.changeText(await screen.findByLabelText("6-digit code"), "000000");
    await fireEvent.press(screen.getByRole("button", { name: "Verify" }));
    expect(await screen.findByText("That code isn't right. Check it and try again.")).toBeTruthy();
    expect(mockStore.size).toBe(0);
  });

  it("rejects a number that is not valid without calling the server", async () => {
    const calls = mockApi({});
    await renderRouter(routes, { initialUrl: "/sign-in" });
    await fireEvent.changeText(await screen.findByLabelText("Mobile number"), "12");
    await fireEvent.press(screen.getByRole("button", { name: "Send 6-digit code" }));
    expect(await screen.findByText("That doesn't look like a valid phone number.")).toBeTruthy();
    expect(calls).toHaveLength(0);
  });

  it("tells the person when they are rate limited", async () => {
    mockApi({
      "POST /v1/auth/otp/request": () => json(429, { code: "rate_limited", message: "x" }),
    });
    await enterNumber();
    expect(
      await screen.findByText("Too many tries. Wait a few minutes and try again."),
    ).toBeTruthy();
  });

  it("waits before allowing a resend", async () => {
    mockApi({ "POST /v1/auth/otp/request": () => json(204) });
    await enterNumber();
    const resend = await screen.findByRole("button", { name: "Resend code" });
    expect(resend.props.accessibilityState).toMatchObject({ disabled: true });
    expect(screen.getByText(/Resend code in \d+s/)).toBeTruthy();
  });

  it("signs in again on relaunch from the saved session", async () => {
    mockStore.set("voyagami.accessToken", "a");
    mockStore.set("voyagami.refreshToken", "r");
    mockApi({ "GET /v1/me": () => json(200, { id: "u1", displayName: "Asha" }) });
    await renderRouter(routes);
    expect(await screen.findByText("Welcome, Asha")).toBeTruthy();
  });

  it("returns to sign in when the saved session is dead", async () => {
    mockStore.set("voyagami.accessToken", "a");
    mockStore.set("voyagami.refreshToken", "r");
    mockApi({
      "GET /v1/me": () => json(401, { code: "unauthenticated", message: "x" }),
      "POST /v1/auth/refresh": () => json(401, { code: "unauthenticated", message: "x" }),
    });
    await renderRouter(routes);
    expect(await screen.findByLabelText("Mobile number")).toBeTruthy();
    await waitFor(() => expect(mockStore.size).toBe(0));
  });

  it("signs out from Profile and clears the saved session", async () => {
    mockStore.set("voyagami.accessToken", "a");
    mockStore.set("voyagami.refreshToken", "r");
    mockApi({
      "GET /v1/me": () => json(200, { id: "u1" }),
      "POST /v1/auth/sign-out": () => json(204),
    });
    await renderRouter(routes, { initialUrl: "/profile" });
    await fireEvent.press(await screen.findByRole("button", { name: "Sign out" }));
    expect(await screen.findByLabelText("Mobile number")).toBeTruthy();
    expect(mockStore.size).toBe(0);
  });
});
