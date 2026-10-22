import tokens from "../../../../design/tokens.json";

export const colors = tokens.color;
export const radius = tokens.radius;
export const hitArea = tokens.hitArea;

export type ColorName = keyof typeof colors;
export type Palette = Record<ColorName, string>;

// Dark tokens are not designed yet beyond the HomeDark board, so dark mirrors light for now.
export const lightPalette: Palette = colors;
export const darkPalette: Palette = colors;
