import { useEffect, type ReactNode } from "react";
import { View } from "react-native";

import { useTheme } from "@/theme";

import { IconButton } from "./IconButton";
import { Text } from "./Text";

export interface ToastProps {
  message: string;
  tone?: "info" | "success";
  onDismiss: () => void;
  /** Screen reader label for the dismiss button. */
  dismissLabel: string;
  dismissIcon: ReactNode;
  /** Milliseconds before it dismisses itself. Pass 0 to keep it until dismissed. Default 5000. */
  durationMs?: number;
}

/**
 * A short message. It is announced politely when it appears, stays long enough to read
 * (5 seconds by default) and always has a dismiss button, so it never traps anyone.
 */
export function Toast({
  message,
  tone = "info",
  onDismiss,
  dismissLabel,
  dismissIcon,
  durationMs = 5000,
}: ToastProps) {
  const theme = useTheme();

  useEffect(() => {
    if (durationMs <= 0) return;
    const id = setTimeout(onDismiss, durationMs);
    return () => clearTimeout(id);
  }, [durationMs, onDismiss, message]);

  return (
    <View
      style={{
        flexDirection: "row",
        alignItems: "center",
        gap: 8,
        paddingLeft: 16,
        paddingRight: 4,
        paddingVertical: 4,
        borderRadius: theme.radius.button,
        backgroundColor: tone === "success" ? theme.colors.matcha : theme.colors.ink,
      }}
    >
      <Text
        accessibilityRole="alert"
        accessibilityLiveRegion="polite"
        style={{ flex: 1 }}
        color={tone === "success" ? "ink" : "white"}
      >
        {message}
      </Text>
      <IconButton accessibilityLabel={dismissLabel} onPress={onDismiss}>
        {dismissIcon}
      </IconButton>
    </View>
  );
}
