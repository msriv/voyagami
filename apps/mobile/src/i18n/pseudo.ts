import { en } from "./en";

const ACCENTS: Record<string, string> = {
  a: "à",
  e: "é",
  i: "î",
  o: "ö",
  u: "ü",
  c: "ç",
  n: "ñ",
  y: "ý",
  A: "Å",
  E: "É",
  I: "Î",
  O: "Ö",
  U: "Û",
  C: "Ç",
  N: "Ñ",
  Y: "Ý",
};

/**
 * Makes a string look foreign and about 40% longer, so clipped or hard-sized layouts show up.
 * {{placeholders}} are left alone so values still fill in. Wrapped in brackets to spot cut-off ends.
 */
export function pseudo(s: string): string {
  const body = s
    .split(/(\{\{[^}]*\}\})/)
    .map((part) =>
      part.startsWith("{{") ? part : part.replace(/[a-zA-Z]/g, (c) => ACCENTS[c] ?? c),
    )
    .join("");
  const pad = "·".repeat(Math.ceil(s.replace(/\{\{[^}]*\}\}/g, "").length * 0.4));
  return `[${body}${pad}]`;
}

type Tree = { [k: string]: string | Tree };
const walk = (t: Tree): Tree =>
  Object.fromEntries(
    Object.entries(t).map(([k, v]) => [k, typeof v === "string" ? pseudo(v) : walk(v)]),
  );

/** The whole English catalog, pseudo-localised. Use it by setting EXPO_PUBLIC_PSEUDO_LOCALE=1. */
export const enXA = walk(en as unknown as Tree);
