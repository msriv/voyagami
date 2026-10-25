import {
  applyOrder,
  displayTime,
  isValidTime,
  parsePin,
  planMove,
  stopsOn,
  type Stop,
} from "./stops";

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

  describe("planMove", () => {
    const day = [
      stop("a", "2026-11-01", "a"),
      stop("b", "2026-11-01", "b"),
      stop("c", "2026-11-01", "c"),
    ];
    const ids = (list: Stop[], date: string) => stopsOn(list, date).map((s) => s.id);
    const at = (id: string) => day.find((s) => s.id === id) as Stop;

    it("moves up before the previous stop", () => {
      const plan = planMove(day, at("b"), { kind: "up" });
      expect(plan?.body).toEqual({ beforeStopId: "a" });
      expect(ids(plan?.local ?? [], "2026-11-01")).toEqual(["b", "a", "c"]);
      expect(planMove(day, at("a"), { kind: "up" })).toBeUndefined();
    });

    it("moves down before the stop after next, or to the end of the day", () => {
      const plan = planMove(day, at("a"), { kind: "down" });
      expect(plan?.body).toEqual({ beforeStopId: "c" });
      expect(ids(plan?.local ?? [], "2026-11-01")).toEqual(["b", "a", "c"]);
      const last = planMove(day, at("b"), { kind: "down" });
      expect(last?.body).toEqual({ date: "2026-11-01" });
      expect(ids(last?.local ?? [], "2026-11-01")).toEqual(["a", "c", "b"]);
      expect(planMove(day, at("c"), { kind: "down" })).toBeUndefined();
    });

    it("moves to a row, clamped to the day", () => {
      const up = planMove(day, at("c"), { kind: "to", index: -5 });
      expect(up?.body).toEqual({ beforeStopId: "a" });
      expect(ids(up?.local ?? [], "2026-11-01")).toEqual(["c", "a", "b"]);
      const end = planMove(day, at("a"), { kind: "to", index: 9 });
      expect(end?.body).toEqual({ date: "2026-11-01" });
      expect(ids(end?.local ?? [], "2026-11-01")).toEqual(["b", "c", "a"]);
      expect(planMove(day, at("b"), { kind: "to", index: 1 })).toBeUndefined();
    });

    it("moves to the end of another day", () => {
      const plan = planMove(day, at("a"), { kind: "day", date: "2026-11-02" });
      expect(plan?.body).toEqual({ date: "2026-11-02" });
      expect(ids(plan?.local ?? [], "2026-11-02")).toEqual(["a"]);
      expect(planMove(day, at("a"), { kind: "day", date: "2026-11-01" })).toBeUndefined();
    });
  });

  it("reads a pin: both blank means none, half or out of range is rejected", () => {
    expect(parsePin("", "")).toEqual({ lat: 0, lng: 0 });
    expect(parsePin(" 21.03 ", "105.85")).toEqual({ lat: 21.03, lng: 105.85 });
    expect(parsePin("21", "")).toBeUndefined();
    expect(parsePin("91", "10")).toBeUndefined();
    expect(parsePin("10", "181")).toBeUndefined();
    expect(parsePin("abc", "10")).toBeUndefined();
  });

  it("applies a new order to one day and leaves other days alone", () => {
    const all = [
      stop("a", "2026-11-01", "a"),
      stop("b", "2026-11-01", "b"),
      stop("c", "2026-11-01", "c"),
      stop("x", "2026-11-02", "a"),
    ];
    const next = applyOrder(all, "2026-11-01", ["c", "a", "b"]);
    expect(stopsOn(next, "2026-11-01").map((s) => s.id)).toEqual(["c", "a", "b"]);
    expect(stopsOn(next, "2026-11-02").map((s) => s.id)).toEqual(["x"]);
  });
});
