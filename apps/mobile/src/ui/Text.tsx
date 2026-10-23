import { Text as RNText, type TextProps as RNTextProps, type TextStyle } from "react-native";

import { useTheme, type ColorName, type FontRole } from "@/theme";

export type TextVariant =
  "display" | "title" | "body" | "bodyStrong" | "caption" | "accent" | "handwriting";

interface VariantStyle {
  font: FontRole;
  size: number;
  lineHeight: number;
  color: ColorName;
  letterSpacing?: number;
}

// Sizes follow the Brand board: Bricolage for headlines and numbers, Manrope for text,
// Instrument Serif italic for one accent word, Caveat for the journal.
const variants: Record<TextVariant, VariantStyle> = {
  display: { font: "display", size: 32, lineHeight: 36, color: "ink", letterSpacing: -0.5 },
  title: { font: "displaySemiBold", size: 22, lineHeight: 28, color: "ink", letterSpacing: -0.2 },
  body: { font: "body", size: 16, lineHeight: 24, color: "ink" },
  bodyStrong: { font: "bodyBold", size: 16, lineHeight: 24, color: "ink" },
  caption: { font: "body", size: 13, lineHeight: 18, color: "muted" },
  accent: { font: "accent", size: 28, lineHeight: 32, color: "ink" },
  handwriting: { font: "handwriting", size: 24, lineHeight: 30, color: "ink" },
};

export interface TextProps extends Omit<RNTextProps, "allowFontScaling" | "maxFontSizeMultiplier"> {
  variant?: TextVariant;
  /** Overrides the variant's colour. Use vermilion for ink-on-light accents only where contrast passes. */
  color?: ColorName;
  /** Marks the text as a heading for screen readers. */
  heading?: boolean;
}

/**
 * The only text component to use for UI text. It always scales with the
 * person's text size: it takes no way to turn scaling off or cap it.
 */
export function Text({ variant = "body", color, heading = false, style, ...rest }: TextProps) {
  const theme = useTheme();
  const v = variants[variant];
  const base: TextStyle = {
    fontFamily: theme.fontFamily[v.font],
    fontSize: v.size,
    lineHeight: v.lineHeight,
    color: theme.colors[color ?? v.color],
    ...(v.letterSpacing === undefined ? null : { letterSpacing: v.letterSpacing }),
  };
  return (
    <RNText accessibilityRole={heading ? "header" : undefined} style={[base, style]} {...rest} />
  );
}
