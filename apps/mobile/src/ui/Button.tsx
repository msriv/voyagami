import {
  ActivityIndicator,
  Pressable,
  type PressableProps,
  type StyleProp,
  type ViewStyle,
} from "react-native";

import { useTheme } from "@/theme";

import { buttonColors, pressFeedback, type ButtonVariant } from "./buttonStyles";
import { Text } from "./Text";
import { useReducedMotion } from "./useReducedMotion";

export interface ButtonProps extends Omit<
  PressableProps,
  "children" | "style" | "role" | "accessibilityRole" | "accessibilityLabel"
> {
  /** The visible text. It is also the screen reader label unless `accessibilityLabel` says more. */
  label: string;
  variant?: ButtonVariant;
  /** Shows a spinner, ignores presses and tells screen readers the button is busy. */
  loading?: boolean;
  /** Fills the width of its container. */
  fullWidth?: boolean;
  accessibilityLabel?: string;
  style?: StyleProp<ViewStyle>;
}

/**
 * A real button: role, label and state (disabled, busy) are always set. At least 44x44 to touch,
 * and the height grows with the text size.
 */
export function Button({
  label,
  variant = "primary",
  loading = false,
  fullWidth = false,
  disabled,
  onPress,
  accessibilityLabel,
  style,
  ...rest
}: ButtonProps) {
  const theme = useTheme();
  const reduceMotion = useReducedMotion();
  const inactive = Boolean(disabled) || loading;
  const colors = buttonColors(theme, variant, Boolean(disabled));
  const min = theme.hitArea.min;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      onPress={onPress}
      hitSlop={8}
      style={({ pressed }) => [
        {
          minHeight: min + 4,
          minWidth: min,
          paddingHorizontal: 20,
          paddingVertical: 12,
          borderRadius: theme.radius.button,
          borderWidth: 1,
          borderColor: colors.border,
          backgroundColor: pressed ? colors.pressedBackground : colors.background,
          alignItems: "center",
          justifyContent: "center",
          flexDirection: "row",
          gap: 8,
          alignSelf: fullWidth ? "stretch" : "flex-start",
          ...pressFeedback(pressed, reduceMotion),
        },
        style,
      ]}
      {...rest}
    >
      {loading ? <ActivityIndicator color={theme.colors[colors.text]} /> : null}
      <Text variant="bodyStrong" color={colors.text}>
        {label}
      </Text>
    </Pressable>
  );
}
