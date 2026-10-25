import { countryName } from "@/trips/destinations";

describe("countryName", () => {
  it("names a country without Intl.DisplayNames, as on Hermes", () => {
    const real = Intl.DisplayNames;
    // @ts-expect-error Hermes does not ship it
    delete Intl.DisplayNames;
    try {
      expect(countryName("VN")).toBe("Vietnam");
      expect(countryName("ZZ")).toBe("ZZ");
    } finally {
      Object.assign(Intl, { DisplayNames: real });
    }
  });
});
