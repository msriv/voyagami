import type { components } from "@voyagami/api-client";

import { i18n } from "@/i18n";

export type Destination = components["schemas"]["Destination"];

// shortcut: a fixed list until place search arrives (VOY-125); then the picker uses search results
// and this file goes away. Hanoi first because it is the pilot city.
export const suggestedDestinations: Destination[] = [
  { name: "Hanoi", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh", localCurrency: "VND" },
  { name: "Ha Long Bay", countryCode: "VN", timeZone: "Asia/Ho_Chi_Minh", localCurrency: "VND" },
  { name: "Bengaluru", countryCode: "IN", timeZone: "Asia/Kolkata", localCurrency: "INR" },
  { name: "Goa", countryCode: "IN", timeZone: "Asia/Kolkata", localCurrency: "INR" },
  { name: "Tokyo", countryCode: "JP", timeZone: "Asia/Tokyo", localCurrency: "JPY" },
  { name: "Kyoto", countryCode: "JP", timeZone: "Asia/Tokyo", localCurrency: "JPY" },
  { name: "Bali", countryCode: "ID", timeZone: "Asia/Makassar", localCurrency: "IDR" },
  { name: "Lisbon", countryCode: "PT", timeZone: "Europe/Lisbon", localCurrency: "EUR" },
  { name: "Paris", countryCode: "FR", timeZone: "Europe/Paris", localCurrency: "EUR" },
  { name: "New York", countryCode: "US", timeZone: "America/New_York", localCurrency: "USD" },
];

/** Country name for an ISO region. Hermes has no Intl.DisplayNames, so names come from i18n, else the code. */
export function countryName(code: string): string {
  return i18n.t(`countries.${code}`, { defaultValue: code });
}
