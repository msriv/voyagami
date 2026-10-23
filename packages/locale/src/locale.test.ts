import { describe, expect, it } from "vitest";

import {
  callingCodeFor,
  defaultClock,
  defaultUnits,
  formatDate,
  formatDistance,
  formatMoney,
  formatTemperature,
  formatTime,
  minorDigits,
  toE164,
} from "./index";

// Intl uses non-breaking spaces and narrow spaces; compare with plain ones.
const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, " ");

describe("minorDigits", () => {
  it.each([
    ["INR", 2],
    ["USD", 2],
    ["EUR", 2],
    ["JPY", 0],
    ["VND", 0],
    ["KWD", 3],
  ])("%s has %i decimals", (code, digits) => {
    expect(minorDigits(code)).toBe(digits);
  });
});

describe("formatMoney", () => {
  it("groups rupees the Indian way", () => {
    expect(plain(formatMoney(16_000_000, "INR", "en-IN"))).toBe("₹1,60,000.00");
  });
  it("shows dollars to the cent", () => {
    expect(formatMoney(109_500, "USD", "en-US")).toBe("$1,095.00");
  });
  it("shows euros the German way", () => {
    expect(plain(formatMoney(109_500, "EUR", "de-DE"))).toBe("1.095,00 €");
  });
  it("treats yen and dong as whole units, not hundredths", () => {
    expect(formatMoney(1200, "JPY", "en-US")).toBe("¥1,200");
    expect(plain(formatMoney(450_000, "VND", "vi-VN"))).toBe("450.000 ₫");
  });
  it("shows three decimals where the currency has them", () => {
    expect(plain(formatMoney(1500, "KWD", "en-US"))).toBe("KWD 1.500");
  });
  it("shows negatives and zero", () => {
    expect(formatMoney(-250, "USD", "en-US")).toBe("-$2.50");
    expect(formatMoney(0, "USD", "en-US")).toBe("$0.00");
  });
  it("refuses fractions of a minor unit and unsafe sizes", () => {
    expect(() => formatMoney(10.5, "USD", "en-US")).toThrow(RangeError);
    expect(() => formatMoney(2 ** 53, "USD", "en-US")).toThrow(RangeError);
  });
});

describe("dates and times", () => {
  const at = new Date("2026-11-21T13:05:00Z");
  it("orders day and month for the viewer", () => {
    expect(formatDate(at, "en-US", { timeZone: "UTC" })).toBe("Nov 21");
    expect(formatDate(at, "en-GB", { timeZone: "UTC" })).toBe("21 Nov");
    expect(formatDate(at, "en-GB", { timeZone: "UTC", withYear: true })).toBe("21 Nov 2026");
  });
  it("shows a time in the trip's zone, not the device's", () => {
    expect(plain(formatTime(at, "en-US", { timeZone: "Asia/Ho_Chi_Minh" }))).toBe("8:05 PM");
    expect(formatTime(at, "en-GB", { timeZone: "UTC" })).toBe("13:05");
  });
  it("lets the person's clock setting win over the locale's", () => {
    expect(formatTime(at, "en-US", { timeZone: "UTC", clock: "24h" })).toBe("13:05");
    expect(plain(formatTime(at, "en-GB", { timeZone: "UTC", clock: "12h" }))).toMatch(/^1:05 pm$/i);
  });
  it("reads the default clock from the locale", () => {
    expect(defaultClock("en-US")).toBe("12h");
    expect(defaultClock("de-DE")).toBe("24h");
  });
});

describe("units", () => {
  it("defaults to miles and Fahrenheit only where people use them", () => {
    expect(defaultUnits("US", "en-US")).toEqual({ distance: "mi", temperature: "f", clock: "12h" });
    expect(defaultUnits("GB", "en-GB")).toMatchObject({ distance: "mi", temperature: "c" });
    expect(defaultUnits("IN", "en-IN")).toMatchObject({ distance: "km", temperature: "c" });
    expect(defaultUnits(null, "en")).toMatchObject({ distance: "km", temperature: "c" });
  });
  it("shows distance in the viewer's unit", () => {
    expect(plain(formatDistance(6200, "km", "en-US"))).toBe("6.2 km");
    expect(plain(formatDistance(6200, "mi", "en-US"))).toBe("3.9 mi");
    expect(plain(formatDistance(6200, "km", "de-DE"))).toBe("6,2 km");
  });
  it("shows temperature in the viewer's unit", () => {
    expect(plain(formatTemperature(22, "c", "en-US"))).toMatch(/^22 ?°C$/);
    expect(plain(formatTemperature(22, "f", "en-US"))).toMatch(/^72 ?°F$/);
  });
});

describe("phone", () => {
  it("reads a national number in the home region", () => {
    expect(toE164("(415) 555-2671", "US")).toBe("+14155552671");
    expect(toE164("98450 12345", "IN")).toBe("+919845012345");
  });
  it("reads a full international number in any region", () => {
    expect(toE164("+44 20 7946 0958", "US")).toBe("+442079460958");
  });
  it("needs a plus sign when the region is unknown", () => {
    expect(toE164("4155552671", null)).toBeNull();
    expect(toE164("+14155552671", null)).toBe("+14155552671");
  });
  it("rejects numbers that are too short", () => {
    expect(toE164("12", "US")).toBeNull();
  });
  it("gives the calling code for a region", () => {
    expect(callingCodeFor("GB")).toBe("44");
    expect(callingCodeFor(null)).toBeNull();
    expect(callingCodeFor("ZZ")).toBeNull();
  });
});
