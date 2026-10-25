import { zonedToUtc } from "./zoned";

describe("zonedToUtc", () => {
  it("handles a fixed-offset zone", () => {
    expect(zonedToUtc("2099-11-19", "09:30", "Asia/Ho_Chi_Minh")).toBe("2099-11-19T02:30:00.000Z");
    expect(zonedToUtc("2099-11-19", "03:00", "Asia/Kolkata")).toBe("2099-11-18T21:30:00.000Z");
  });

  it("follows daylight saving", () => {
    expect(zonedToUtc("2026-01-15", "12:00", "America/New_York")).toBe("2026-01-15T17:00:00.000Z");
    expect(zonedToUtc("2026-07-15", "12:00", "America/New_York")).toBe("2026-07-15T16:00:00.000Z");
  });
});
