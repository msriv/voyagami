import tokens from "../../../../../design/tokens.json";
import { fontAssets, fontFamily } from "../fonts";
import { colors, hitArea, lightPalette } from "../tokens";

describe("theme", () => {
  it("takes every colour from design/tokens.json", () => {
    expect(lightPalette).toEqual(tokens.color);
    expect(colors.indigo).toBe("#2D3E8C");
    expect(colors.washi).toBe("#F4F0E6");
  });

  it("keeps the 44 point minimum hit area", () => {
    expect(hitArea.min).toBe(44);
  });

  it("only names font families that are loaded", () => {
    for (const family of Object.values(fontFamily)) {
      expect(Object.keys(fontAssets)).toContain(family);
    }
  });
});
