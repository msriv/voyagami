import type { ReactNode } from "react";
import { Pressable, type PressableProps, type StyleProp, type ViewStyle } from "react-native";

import { useTheme } from "@/theme";

import { buttonColors, pressFeedback, type ButtonVariant } from "./buttonStyles";
import { useReducedMotion } from "./useReducedMotion";

export interface IconButtonProps extends Omit<
  PressableProps,
  "children" | "style" | "role" | "accessibilityRole" | "accessibilityLabel"
> {
  /** What the button does, for screen readers. Required because there is no visible text. */
  accessibilityLabel: string;
  /** The icon. Hide it from screen readers; the label speaks for the button. */
  children: ReactNode;
  variant?: ButtonVariant;
  style?: StyleProp<ViewStyle>;
}

/** An icon-only button: a 44x44 hit area around the icon, with a required label. */
export function IconButton({
  accessibilityLabel,
  children,
  variant = "ghost",
  disabled,
  onPress,
  style,
  ...rest
}: IconButtonProps) {
  const theme = useTheme();
  const reduceMotion = useReducedMotion();
  const colors = buttonColors(theme, variant, Boolean(disabled));
  const size = theme.hitArea.min;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      accessibilityState={{ disabled: Boolean(disabled) }}
      disabled={disabled}
      onPress={onPress}
      hitSlop={4}
      style={({ pressed }) => [
        {
          width: size,
          height: size,
          borderRadius: theme.radius.button,
          borderWidth: 1,
          borderColor: colors.border,
          backgroundColor: pressed ? colors.pressedBackground : colors.background,
          alignItems: "center",
          justifyContent: "center",
          ...pressFeedback(pressed, reduceMotion),
        },
        style,
      ]}
      {...rest}
    >
      {children}
    </Pressable>
  );
}
