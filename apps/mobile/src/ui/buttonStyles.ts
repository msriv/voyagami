import type { ViewStyle } from "react-native";

import type { ColorName, Theme } from "@/theme";

export type ButtonVariant = "primary" | "accent" | "secondary" | "ghost" | "dark";

export interface ButtonColors {
  background: string;
  text: ColorName;
  border: string;
  pressedBackground: string;
}

/**
 * Colours per variant. White on vermilion fails contrast, so accent buttons use ink text
 * (Brand board). Contrast: primary 9.6:1, accent 4.6:1, secondary 18:1, ghost 8.5:1.
 */
export function buttonColors(
  theme: Theme,
  variant: ButtonVariant,
  disabled: boolean,
): ButtonColors {
  const c = theme.colors;
  if (disabled) {
    return {
      background: c.chipOff,
      text: "muted",
      border: c.chipOff,
      pressedBackground: c.chipOff,
    };
  }
  switch (variant) {
    case "primary":
      return { background: c.indigo, text: "white", border: c.indigo, pressedBackground: c.indigo };
    case "dark":
      return { background: c.ink, text: "white", border: c.ink, pressedBackground: c.ink };
    case "accent":
      return {
        background: c.vermilion,
        text: "ink",
        border: c.vermilion,
        pressedBackground: "#CC3E27",
      };
    case "secondary":
      return { background: c.white, text: "ink", border: c.line, pressedBackground: c.washi };
    case "ghost":
      return {
        background: "transparent",
        text: "indigo",
        border: "transparent",
        pressedBackground: c.line,
      };
  }
}

/** Pressed controls dim to this opacity (primary and accent) so the press shows without a colour that is not a token. */
export const PRESSED_OPACITY = 0.85;

/** Pressing scales a control to this size, unless reduce motion is on. */
export const PRESSED_SCALE = 0.97;

/** Style for a control while it is pressed. With reduce motion on there is no scale, only the dimming. */
export function pressFeedback(pressed: boolean, reduceMotion: boolean): ViewStyle {
  return {
    opacity: pressed ? PRESSED_OPACITY : 1,
    transform: pressed && !reduceMotion ? [{ scale: PRESSED_SCALE }] : [],
  };
}
