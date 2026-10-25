import { renderHook, waitFor } from "@testing-library/react-native";

import { useEntitlement } from "..";

const mockApi = { GET: jest.fn(), unwrap: jest.fn((p: Promise<unknown>) => p) };
jest.mock("@/auth", () => ({ useAuth: () => ({ api: mockApi, status: "signedIn" }) }));

const answer = {
  plan: "free",
  features: { multiCurrency: { allowed: false, limit: 0, used: 0, needsPlus: true } },
};

beforeEach(() => mockApi.GET.mockReset());

describe("useEntitlement", () => {
  it("reflects the server's answer", async () => {
    mockApi.GET.mockResolvedValue(answer);
    const { result } = await renderHook(() => useEntitlement("multiCurrency"));
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current).toMatchObject({ allowed: false, needsPlus: true });
    expect(mockApi.GET).toHaveBeenCalledWith("/v1/me/entitlements");
  });

  it("asks about the trip when given a trip id", async () => {
    mockApi.GET.mockResolvedValue({
      plan: "pass",
      features: { multiCurrency: { allowed: true, limit: 0, used: 0, needsPlus: false } },
    });
    const { result } = await renderHook(() => useEntitlement("multiCurrency", "t1"));
    await waitFor(() => expect(result.current.allowed).toBe(true));
    expect(mockApi.GET).toHaveBeenCalledWith("/v1/trips/{tripId}/entitlements", {
      params: { path: { tripId: "t1" } },
    });
  });

  it("stays closed when the request fails", async () => {
    mockApi.GET.mockRejectedValue(new Error("offline"));
    const { result } = await renderHook(() => useEntitlement("multiCurrency"));
    await waitFor(() => expect(mockApi.GET).toHaveBeenCalled());
    expect(result.current).toMatchObject({ loaded: false, allowed: false });
  });
});
