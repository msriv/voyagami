import type { ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";

import { useTheme } from "@/theme";

export interface CardProps {
  children: ReactNode;
  /** Washi card on white by default; "tinted" uses the washi background for use on white. */
  tone?: "plain" | "tinted";
  style?: StyleProp<ViewStyle>;
}

/** A washi card: radius 24, a thin line border, no shadow. Not tappable; wrap content in a Button for that. */
export function Card({ children, tone = "plain", style }: CardProps) {
  const theme = useTheme();
  return (
    <View
      style={[
        {
          borderRadius: theme.radius.card,
          borderWidth: 1,
          borderColor: theme.colors.line,
          backgroundColor: tone === "plain" ? theme.colors.white : theme.colors.washi,
          padding: 16,
        },
        style,
      ]}
    >
      {children}
    </View>
  );
}
