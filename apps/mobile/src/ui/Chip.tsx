import { Pressable, type PressableProps } from "react-native";

import { useTheme } from "@/theme";

import { pressFeedback } from "./buttonStyles";
import { Text } from "./Text";
import { useReducedMotion } from "./useReducedMotion";

export interface ChipProps extends Omit<
  PressableProps,
  "children" | "style" | "role" | "accessibilityRole" | "accessibilityState"
> {
  label: string;
  selected: boolean;
}

/**
 * A selectable chip. Selected state is shown by fill and weight, and is also
 * announced to screen readers, so colour is never the only signal.
 */
export function Chip({ label, selected, disabled, onPress, ...rest }: ChipProps) {
  const theme = useTheme();
  const reduceMotion = useReducedMotion();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected, disabled: Boolean(disabled) }}
      disabled={disabled}
      onPress={onPress}
      hitSlop={4}
      style={({ pressed }) => ({
        minHeight: theme.hitArea.min,
        minWidth: theme.hitArea.min,
        paddingHorizontal: 16,
        paddingVertical: 8,
        borderRadius: theme.radius.pill,
        borderWidth: 1,
        borderColor: selected ? theme.colors.indigo : theme.colors.chipOff,
        backgroundColor: selected ? theme.colors.indigo : theme.colors.white,
        alignItems: "center",
        justifyContent: "center",
        opacity: disabled ? 0.5 : 1,
        ...pressFeedback(pressed, reduceMotion),
      })}
      {...rest}
    >
      <Text variant={selected ? "bodyStrong" : "body"} color={selected ? "white" : "ink"}>
        {label}
      </Text>
    </Pressable>
  );
}
