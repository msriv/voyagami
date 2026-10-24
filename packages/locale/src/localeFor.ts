/** The device language with the person's home region, for example "en" and "IN" give "en-IN". Dates, numbers and money follow the region. */
export function localeFor(languageTag: string, region: string | null): string {
  if (!region) return languageTag;
  try {
    const l = new Intl.Locale(languageTag);
    return new Intl.Locale(l.language, {
      region,
      ...(l.script ? { script: l.script } : null),
    }).toString();
  } catch {
    return languageTag;
  }
}
