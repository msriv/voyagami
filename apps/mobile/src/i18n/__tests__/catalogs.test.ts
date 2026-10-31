import { en } from "../en";
import { hi, REQUIRED_NAMESPACES } from "../hi";

type Tree = { [k: string]: string | Tree };

/** Every leaf as `a.b.c` -> string. */
function leaves(tree: Tree, prefix = ""): Record<string, string> {
  return Object.entries(tree).reduce<Record<string, string>>((out, [k, v]) => {
    const key = prefix ? `${prefix}.${k}` : k;
    return typeof v === "string" ? { ...out, [key]: v } : { ...out, ...leaves(v, key) };
  }, {});
}

const placeholders = (s: string) => [...s.matchAll(/\{\{\s*(\w+)\s*\}\}/g)].map((m) => m[1]).sort();
const english = leaves(en as unknown as Tree);

describe.each([["hi", hi]])("%s catalog", (_name, catalog) => {
  const own = leaves(catalog as unknown as Tree);

  it("has no key English does not have", () => {
    const stray = Object.keys(own).filter((k) => !(k in english));
    expect(stray).toEqual([]);
  });

  it("keeps the same {{placeholders}} as English", () => {
    const wrong = Object.entries(own)
      .filter(([k, v]) => placeholders(v).join() !== placeholders(english[k] ?? "").join())
      .map(([k]) => k);
    expect(wrong).toEqual([]);
  });

  it("is complete for the required namespaces", () => {
    const missing = Object.keys(english).filter(
      (k) => REQUIRED_NAMESPACES.some((n) => k.startsWith(`${n}.`)) && !(k in own),
    );
    expect(missing).toEqual([]);
  });
});
