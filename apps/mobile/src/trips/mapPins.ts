import { stopsOn, type Stop } from "./stops";

export type Pin = {
  id: string;
  number: number;
  name: string;
  lat: number;
  lng: number;
  date: string;
  startLocal?: string;
  note?: string;
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
              ...(s.startLocal ? { startLocal: s.startLocal } : null),
              ...(s.note ? { note: s.note } : null),
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

/** The link that opens directions to a pin in the phone's own maps app. */
export function directionsUrl(os: string, lat: number, lng: number, name: string): string {
  const q = `${lat},${lng}`;
  if (os === "ios") return `https://maps.apple.com/?daddr=${q}&q=${encodeURIComponent(name)}`;
  if (os === "android") return `geo:0,0?q=${q}(${encodeURIComponent(name)})`;
  return `https://www.google.com/maps/dir/?api=1&destination=${q}`;
}
