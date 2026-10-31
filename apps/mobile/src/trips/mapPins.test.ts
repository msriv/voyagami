import { daysWithStops, directionsUrl, mapPins } from "./mapPins";
import type { Stop } from "./stops";

const stop = (id: string, date: string, order: string, lat?: number, lng?: number): Stop => ({
  id,
  tripId: "t",
  date,
  order,
  place: lat === undefined ? { name: id } : { name: id, lat, lng: lng as number },
  timeZone: "UTC",
});

const stops = [
  stop("b", "2099-01-01", "b", 2, 2),
  stop("a", "2099-01-01", "a", 1, 1),
  stop("nopin", "2099-01-01", "c", 0, 0),
  stop("d", "2099-01-02", "a", 4, 4),
];

describe("mapPins", () => {
  it("numbers pins by plan order and skips stops without a pin", () => {
    expect(mapPins(stops, ["2099-01-01"]).map((p) => [p.id, p.number])).toEqual([
      ["a", 1],
      ["b", 2],
    ]);
  });
  it("covers several days in order", () => {
    expect(mapPins(stops, ["2099-01-01", "2099-01-02"]).map((p) => p.id)).toEqual(["a", "b", "d"]);
  });
  it("lists days that have stops", () => {
    expect(daysWithStops(stops)).toEqual(["2099-01-01", "2099-01-02"]);
  });
});

describe("directionsUrl", () => {
  it("uses the maps app of the platform", () => {
    expect(directionsUrl("ios", 1, 2, "A b")).toBe("https://maps.apple.com/?daddr=1,2&q=A%20b");
    expect(directionsUrl("android", 1, 2, "A")).toBe("geo:0,0?q=1,2(A)");
    expect(directionsUrl("web", 1, 2, "A")).toContain("destination=1,2");
  });
});
