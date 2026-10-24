// First guess at a person's currency from their home region, before they confirm it (VOY-111).
// Regions not listed return null, and the person picks one.
const byRegion: Record<string, string> = {
  AU: "AUD",
  BD: "BDT",
  BR: "BRL",
  CA: "CAD",
  CH: "CHF",
  CN: "CNY",
  GB: "GBP",
  HK: "HKD",
  ID: "IDR",
  IN: "INR",
  JP: "JPY",
  KR: "KRW",
  LK: "LKR",
  MX: "MXN",
  MY: "MYR",
  NP: "NPR",
  NZ: "NZD",
  PH: "PHP",
  PK: "PKR",
  SA: "SAR",
  SG: "SGD",
  TH: "THB",
  TR: "TRY",
  US: "USD",
  AE: "AED",
  VN: "VND",
  ZA: "ZAR",
};
const euro = new Set([
  "AT",
  "BE",
  "CY",
  "DE",
  "EE",
  "ES",
  "FI",
  "FR",
  "GR",
  "HR",
  "IE",
  "IT",
  "LT",
  "LU",
  "LV",
  "MT",
  "NL",
  "PT",
  "SI",
  "SK",
]);

/** The ISO 4217 code most people in a region use, or null when we are not sure. */
export function defaultCurrency(region: string | null): string | null {
  const r = region?.toUpperCase() ?? "";
  return euro.has(r) ? "EUR" : (byRegion[r] ?? null);
}
