import {
  getCountryCallingCode,
  parsePhoneNumberFromString,
  type CountryCode,
} from "libphonenumber-js";

/**
 * Turns what the person typed into E.164 (for example +14155550123), or null if it is not a
 * valid number. Numbers without a leading "+" are read in `region`, the device's region.
 */
export function toE164(input: string, region: string | null): string | null {
  const parsed = parsePhoneNumberFromString(
    input.trim(),
    (region ?? undefined) as CountryCode | undefined,
  );
  return parsed?.isValid() ? parsed.number : null;
}

/** The calling code for a region, for the hint under the field, or null when the region is unknown. */
export function callingCodeFor(region: string | null): string | null {
  try {
    return region ? getCountryCallingCode(region as CountryCode) : null;
  } catch {
    return null;
  }
}
