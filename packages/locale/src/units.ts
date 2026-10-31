import type { Clock } from "./datetime";
import { defaultClock } from "./datetime";

export type DistanceUnit = "km" | "mi";
export type TemperatureUnit = "c" | "f";

/** Matches the `units` object on the user document (see docs/data-model.md). */
export interface Units {
  distance: DistanceUnit;
  temperature: TemperatureUnit;
  clock: Clock;
}

/** Every value a person can pick for each unit setting, for building pickers. */
export const unitOptions = {
  distance: ["km", "mi"],
  temperature: ["c", "f"],
  clock: ["12h", "24h"],
} as const satisfies { [K in keyof Units]?: readonly Units[K][] };

// Regions whose roads are signed in miles, and whose weather is given in Fahrenheit.
const mileRegions = new Set(["US", "GB", "LR", "MM"]);
const fahrenheitRegions = new Set(["US", "BS", "BZ", "KY", "LR", "PW"]);

/** First guess at a person's units from their home region and locale, before they change it. */
export function defaultUnits(region: string | null, locale: string): Units {
  const r = region?.toUpperCase() ?? "";
  return {
    distance: mileRegions.has(r) ? "mi" : "km",
    temperature: fahrenheitRegions.has(r) ? "f" : "c",
    clock: defaultClock(locale),
  };
}

const metersPerMile = 1609.344;

/** A distance given in metres, shown in the viewer's unit with the platform's unit name, for example "6.2 km" or "3.9 mi". */
export function formatDistance(meters: number, distance: DistanceUnit, locale: string): string {
  const value = distance === "mi" ? meters / metersPerMile : meters / 1000;
  return new Intl.NumberFormat(locale, {
    style: "unit",
    unit: distance === "mi" ? "mile" : "kilometer",
    unitDisplay: "short",
    maximumFractionDigits: 1,
  }).format(value);
}

/** A temperature given in degrees Celsius, shown in the viewer's unit. */
export function formatTemperature(
  celsius: number,
  temperature: TemperatureUnit,
  locale: string,
): string {
  const value = temperature === "f" ? (celsius * 9) / 5 + 32 : celsius;
  return new Intl.NumberFormat(locale, {
    style: "unit",
    unit: temperature === "f" ? "fahrenheit" : "celsius",
    maximumFractionDigits: 0,
  }).format(value);
}

/** A size in bytes, shown in kilobytes below a megabyte and megabytes above, with the platform's unit name. */
export function formatBytes(bytes: number, locale: string): string {
  const mega = bytes >= 1_000_000;
  return new Intl.NumberFormat(locale, {
    style: "unit",
    unit: mega ? "megabyte" : "kilobyte",
    unitDisplay: "short",
    maximumFractionDigits: mega ? 1 : 0,
  }).format(mega ? bytes / 1_000_000 : Math.max(1, bytes / 1000));
}
