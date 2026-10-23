import { callingCodeFor, toE164 } from "../phone";

describe("toE164", () => {
  it("reads a national number in the device region", () => {
    expect(toE164("(415) 555-2671", "US")).toBe("+14155552671");
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
});

describe("callingCodeFor", () => {
  it("returns the code for a region and null for none or nonsense", () => {
    expect(callingCodeFor("GB")).toBe("44");
    expect(callingCodeFor(null)).toBeNull();
    expect(callingCodeFor("ZZ")).toBeNull();
  });
});
