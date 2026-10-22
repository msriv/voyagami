import {
  BricolageGrotesque_600SemiBold,
  BricolageGrotesque_700Bold,
} from "@expo-google-fonts/bricolage-grotesque";
import { Caveat_600SemiBold } from "@expo-google-fonts/caveat";
import { InstrumentSerif_400Regular_Italic } from "@expo-google-fonts/instrument-serif";
import {
  Manrope_400Regular,
  Manrope_600SemiBold,
  Manrope_700Bold,
} from "@expo-google-fonts/manrope";

/** Font files to load at start-up, keyed by the family name used in styles. */
export const fontAssets = {
  BricolageGrotesque_600SemiBold,
  BricolageGrotesque_700Bold,
  Manrope_400Regular,
  Manrope_600SemiBold,
  Manrope_700Bold,
  InstrumentSerif_400Regular_Italic,
  Caveat_600SemiBold,
};

/** Family names to use in `fontFamily`. Display: headings. Body: text. Accent: italic words. Handwriting: journal. */
export const fontFamily = {
  display: "BricolageGrotesque_700Bold",
  displaySemiBold: "BricolageGrotesque_600SemiBold",
  body: "Manrope_400Regular",
  bodySemiBold: "Manrope_600SemiBold",
  bodyBold: "Manrope_700Bold",
  accent: "InstrumentSerif_400Regular_Italic",
  handwriting: "Caveat_600SemiBold",
} as const satisfies Record<string, keyof typeof fontAssets>;

export type FontRole = keyof typeof fontFamily;
