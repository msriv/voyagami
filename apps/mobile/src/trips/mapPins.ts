import { stopsOn, type Stop } from "./stops";

export type Pin = {
  id: string;
  number: number;
  name: string;
  lat: number;
  lng: number;
  date: string;
};

/** The API stores 0,0 for "no pin". */
function hasPin(s: Stop): boolean {
  const { lat, lng } = s.place;
  return typeof lat === "number" && typeof lng === "number" && !(lat === 0 && lng === 0);
}

/**
 * Pins for the map, numbered by plan order within each day. Stops without a
 * pin are skipped but still count toward the number, so the map matches the list.
 */
export function mapPins(stops: Stop[], dates: string[]): Pin[] {
  return dates.flatMap((date) =>
    stopsOn(stops, date).flatMap((s, i) =>
      hasPin(s)
        ? [
            {
              id: s.id,
              number: i + 1,
              name: s.place.name ?? "",
              lat: s.place.lat as number,
              lng: s.place.lng as number,
              date,
            },
          ]
        : [],
    ),
  );
}

/** The days that have at least one stop, in order. */
export function daysWithStops(stops: Stop[]): string[] {
  return [...new Set(stops.map((s) => s.date))].sort();
}
