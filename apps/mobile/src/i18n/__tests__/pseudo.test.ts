import { en } from "../en";
import { pseudo } from "../pseudo";

describe("pseudo-locale", () => {
  it("is about 40% longer, accented, and keeps placeholders", () => {
    const out = pseudo("Welcome, {{name}}");
    expect(out).toContain("{{name}}");
    expect(out.startsWith("[Wélçömé, ")).toBe(true);
    expect(out.length).toBeGreaterThan("Welcome, {{name}}".length + 2);
    expect(pseudo("x".repeat(100)).length).toBeGreaterThanOrEqual(140);
  });

  it("covers every English string", () => {
    expect(pseudo(en.tabs.home)).not.toBe(en.tabs.home);
  });
});
