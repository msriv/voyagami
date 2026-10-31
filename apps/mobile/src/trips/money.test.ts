import {
  convertMinor,
  parseAmount,
  rateFromMajor,
  rateToMajor,
  todayIn,
  type RateRatio,
} from "./money";

describe("parseAmount", () => {
  it("reads two-decimal currencies", () => {
    expect(parseAmount("12.5", "USD")).toBe(1250);
    expect(parseAmount("12,50", "EUR")).toBe(1250);
    expect(parseAmount("7", "USD")).toBe(700);
    expect(parseAmount("0.05", "USD")).toBe(5);
  });

  it("reads zero-decimal currencies", () => {
    expect(parseAmount("50000", "VND")).toBe(50000);
    expect(parseAmount("1200", "JPY")).toBe(1200);
    expect(parseAmount("12.5", "JPY")).toBeUndefined();
  });

  it("refuses zero, junk and too many decimals", () => {
    for (const bad of ["", "0", "0.00", "-5", "abc", "1.234", "1.2.3"]) {
      expect(parseAmount(bad, "USD")).toBeUndefined();
    }
  });
});

describe("todayIn", () => {
  it("uses the trip's zone, not the phone's", () => {
    const at = new Date("2026-11-01T20:00:00Z");
    expect(todayIn("Asia/Ho_Chi_Minh", at)).toBe("2026-11-02");
    expect(todayIn("America/Los_Angeles", at)).toBe("2026-11-01");
  });
});

describe("rates", () => {
  it("turns a typed rate into the minor-unit ratio", () => {
    expect(rateFromMajor("25000", "USD", "VND")).toEqual({ numerator: 250, denominator: 1 });
    expect(rateFromMajor("0.0085", "INR", "USD")).toEqual({ numerator: 17, denominator: 2000 });
    expect(rateFromMajor("150,5", "USD", "JPY")).toEqual({ numerator: 301, denominator: 200 });
  });
  it("refuses junk and zero", () => {
    for (const t of ["", "abc", "0", "0.0", "-5", "1.2.3"]) {
      expect(rateFromMajor(t, "USD", "VND")).toBeUndefined();
    }
  });
  it("converts and shows rates", () => {
    const r = rateFromMajor("25000", "USD", "VND") as RateRatio;
    expect(convertMinor(1999, r)).toBe(499750);
    expect(rateToMajor(r, "USD", "VND")).toBe(25000);
  });
});
