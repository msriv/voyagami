import { countdown, partitionTrips, type Trip } from "./summary";

const trip = (id: string, dates: Trip["dates"], status: Trip["status"] = "planned"): Trip => ({
  id,
  title: id,
  ownerId: "u",
  destinations: [],
  dates,
  timeZone: "UTC",
  settlementCurrency: "USD",
  status,
  role: "owner",
});

describe("trip summary", () => {
  const today = "2026-11-10";
  it("splits upcoming from past and keeps order", () => {
    const a = trip("a", { start: "2026-11-19", end: "2026-11-27" });
    const b = trip("b", { start: "2026-01-01", end: "2026-01-05" });
    const c = trip("c", { flexible: { days: 5 } });
    const d = trip("d", { start: "2026-12-01", end: "2026-12-03" }, "past");
    const e = trip("e", { start: "2026-11-08", end: "2026-11-12" });
    const { upcoming, past } = partitionTrips([a, b, c, d, e], today);
    expect(upcoming.map((t) => t.id)).toEqual(["a", "c", "e"]);
    expect(past.map((t) => t.id)).toEqual(["b", "d"]);
  });
  it("counts down to the start", () => {
    expect(countdown(trip("a", { start: "2026-11-19", end: "2026-11-27" }), today)).toEqual({
      kind: "in",
      days: 9,
    });
    expect(countdown(trip("a", { start: "2026-11-10", end: "2026-11-12" }), today)).toEqual({
      kind: "ongoing",
    });
    expect(countdown(trip("a", { flexible: { days: 3 } }), today)).toEqual({ kind: "none" });
  });
});
