import { ESLint } from "eslint";
import { describe, expect, it } from "vitest";

// The repo's own lint config, checked against code that should and should not pass.
const eslint = new ESLint({ cwd: new URL("../../..", import.meta.url).pathname });

async function messages(code: string, filePath = "apps/mobile/src/example.ts") {
  const [result] = await eslint.lintText(code, { filePath });
  return result?.messages.map((m) => m.message) ?? [];
}

describe("hard-coded locale values are blocked in app code", () => {
  it.each([
    ['export const a = "₹999";', "currency symbol"],
    ["export const a = `€${x}`;", "currency symbol"],
    ['export const a = "$29.99";', "dollar amount"],
    ['export const a = "6 km";', "distance unit"],
    ["export const a = `${x} mi`;", "distance unit"],
    ['export const a = "+91 98450 12345";', "calling code"],
    ['export const a = "DD/MM/YYYY";', "date format"],
    ["export const a = x.toFixed(2);", "by hand"],
    ["export const a = x.toLocaleString();", "by hand"],
    ['export const a = new Intl.NumberFormat("en");', "Intl directly"],
  ])("%s", async (code, what) => {
    expect((await messages(code)).join(" ")).toContain(what);
  });

  it("allows ordinary code and text", async () => {
    expect(
      await messages('export const a = "Mobile number"; export const b = `Hello ${a}`;'),
    ).toEqual([]);
  });

  it("leaves the locale module and tests alone", async () => {
    expect(await messages('export const a = "₹999";', "packages/locale/src/example.ts")).toEqual(
      [],
    );
    expect(
      await messages('export const a = "₹999";', "apps/mobile/src/__tests__/example.ts"),
    ).toEqual([]);
    expect(
      await messages('export const a = "₹999";', "packages/api-client/src/client.test.ts"),
    ).toEqual([]);
  });
});
