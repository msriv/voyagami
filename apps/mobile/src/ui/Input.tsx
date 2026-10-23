import { useId, useState } from "react";
import { TextInput, View, type TextInputProps } from "react-native";

import { useTheme } from "@/theme";

import { Text } from "./Text";

export interface InputProps extends Omit<
  TextInputProps,
  | "style"
  | "accessibilityLabel"
  | "accessibilityState"
  | "allowFontScaling"
  | "maxFontSizeMultiplier"
> {
  /** Always visible above the field; also the screen reader label. */
  label: string;
  /** Shown below the field in place of the hint. Marks the field invalid. */
  error?: string | undefined;
  hint?: string | undefined;
}

/** A text input with a visible label and an error line. Text scales; the box grows with it. */
export function Input({
  label,
  error,
  hint,
  editable = true,
  onFocus,
  onBlur,
  ...rest
}: InputProps) {
  const theme = useTheme();
  const [focused, setFocused] = useState(false);
  const helperId = useId();
  const invalid = error !== undefined && error !== "";
  const helper = invalid ? error : hint;

  return (
    <View style={{ gap: 6 }}>
      <Text variant="bodyStrong">{label}</Text>
      <TextInput
        accessibilityLabel={label}
        accessibilityHint={helper}
        accessibilityState={{ disabled: !editable }}
        aria-invalid={invalid}
        aria-describedby={helper ? helperId : undefined}
        editable={editable}
        placeholderTextColor={theme.colors.muted}
        onFocus={(e) => {
          setFocused(true);
          onFocus?.(e);
        }}
        onBlur={(e) => {
          setFocused(false);
          onBlur?.(e);
        }}
        style={{
          minHeight: theme.hitArea.min,
          paddingHorizontal: 14,
          paddingVertical: 10,
          borderRadius: theme.radius.button,
          borderWidth: focused || invalid ? 2 : 1,
          borderColor: invalid
            ? theme.colors.vermilion
            : focused
              ? theme.colors.indigo
              : theme.colors.chipOff,
          backgroundColor: editable ? theme.colors.white : theme.colors.line,
          color: theme.colors.ink,
          fontFamily: theme.fontFamily.body,
          fontSize: 16,
        }}
        {...rest}
      />
      {helper ? (
        <Text
          nativeID={helperId}
          variant="caption"
          color={invalid ? "ink" : "muted"}
          accessibilityLiveRegion={invalid ? "polite" : "none"}
        >
          {helper}
        </Text>
      ) : null}
    </View>
  );
}
