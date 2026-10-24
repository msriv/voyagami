import { daysBetween, daysInRange, monthCells, pickDay } from "./dates";

describe("trip dates", () => {
  it("counts days inclusive", () => {
    expect(daysBetween("2026-11-19", "2026-11-27")).toBe(9);
    expect(daysBetween("2026-02-28", "2026-03-01")).toBe(2);
  });
  it("picks a start, then an end, then starts again", () => {
    const a = pickDay({}, "2026-11-19");
    expect(a).toEqual({ start: "2026-11-19" });
    const b = pickDay(a, "2026-11-27");
    expect(b).toEqual({ start: "2026-11-19", end: "2026-11-27" });
    expect(pickDay(b, "2026-12-01")).toEqual({ start: "2026-12-01" });
    expect(pickDay(a, "2026-11-10")).toEqual({ start: "2026-11-10" });
  });
  it("lines day 1 up under its weekday", () => {
    // 1 Nov 2026 is a Sunday.
    expect(monthCells(2026, 10, 0)[0]).toBe("2026-11-01");
    expect(monthCells(2026, 10, 1).slice(0, 6)).toEqual([null, null, null, null, null, null]);
    expect(monthCells(2026, 10, 1)).toHaveLength(6 + 30);
  });
  it("lists every day in a range and caps long ones", () => {
    expect(daysInRange("2026-02-27", "2026-03-02")).toEqual([
      "2026-02-27",
      "2026-02-28",
      "2026-03-01",
      "2026-03-02",
    ]);
    expect(daysInRange("2026-01-01", "2026-12-31", 5)).toHaveLength(5);
  });
});
