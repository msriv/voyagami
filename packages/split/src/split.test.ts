import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import { SplitError, split, type SplitMode } from "./index";

interface Vector {
  name: string;
  total: number;
  mode: SplitMode;
  parts: [string, number][];
  want?: number[];
  error?: boolean;
}

const { cases } = JSON.parse(readFileSync(new URL("../vectors.json", import.meta.url), "utf8")) as {
  cases: Vector[];
};

describe("shared vectors (also run by the Go tests)", () => {
  for (const c of cases) {
    it(c.name, () => {
      const parts = c.parts.map(([uid, value]) => ({ uid, value }));
      if (c.error) {
        expect(() => split(c.total, c.mode, parts)).toThrow(SplitError);
        return;
      }
      const got = split(c.total, c.mode, parts);
      expect(got.map((a) => a.amountMinor)).toEqual(c.want);
      expect(got.map((a) => a.uid)).toEqual(parts.map((p) => p.uid));
    });
  }
});

describe("properties", () => {
  it("adds up to the total and stays within one unit of the exact share", () => {
    let seed = 12345;
    const next = (n: number) => {
      seed = (seed * 1103515245 + 12345) % 2147483648;
      return seed % n;
    };
    for (let k = 0; k < 2000; k++) {
      const n = 1 + next(9);
      const total = next(1_000_000_000);
      const parts = Array.from({ length: n }, (_, i) => ({ uid: `u${i}`, value: 1 + next(20) }));
      const sumW = parts.reduce((a, p) => a + p.value, 0);
      const got = split(total, "shares", parts);
      expect(got.reduce((a, x) => a + x.amountMinor, 0)).toBe(total);
      got.forEach((a, i) => {
        const low = Math.floor((total * (parts[i]?.value ?? 0)) / sumW);
        expect(a.amountMinor).toBeGreaterThanOrEqual(low);
        expect(a.amountMinor).toBeLessThanOrEqual(low + 1);
      });
    }
  });

  it("rejects amounts that are not whole minor units", () => {
    expect(() => split(10.5, "equal", [{ uid: "a", value: 0 }])).toThrow(SplitError);
  });
});
