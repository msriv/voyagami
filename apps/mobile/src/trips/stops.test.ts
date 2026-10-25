import { displayTime, isValidTime, stopsOn, type Stop } from "./stops";

const stop = (id: string, date: string, order: string): Stop => ({
  id,
  tripId: "t",
  date,
  order,
  place: { name: id },
  timeZone: "UTC",
});

describe("stops", () => {
  it("keeps one day's stops in plan order", () => {
    const all = [
      stop("c", "2026-11-02", "b"),
      stop("a", "2026-11-01", "a"),
      stop("b", "2026-11-02", "a"),
    ];
    expect(stopsOn(all, "2026-11-02").map((s) => s.id)).toEqual(["b", "c"]);
    expect(stopsOn(all, "2026-11-03")).toEqual([]);
  });

  it("accepts blank or 24-hour times only", () => {
    expect(isValidTime("")).toBe(true);
    expect(isValidTime("09:30")).toBe(true);
    expect(isValidTime("24:00")).toBe(false);
    expect(isValidTime("9:30")).toBe(false);
    expect(isValidTime("09:60")).toBe(false);
  });

  it("shows times in the viewer's clock", () => {
    expect(displayTime("14:05", "en-US")).toMatch(/2:05\s?PM/);
    expect(displayTime("14:05", "en-GB")).toBe("14:05");
    expect(displayTime("", "en-US")).toBe("");
  });
});
