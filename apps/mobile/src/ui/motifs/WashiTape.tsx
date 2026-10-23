import { View, type StyleProp, type ViewStyle } from "react-native";

import { useTheme } from "@/theme";

export interface WashiTapeProps {
  width?: number;
  height?: number;
  /** Degrees. Default -8. */
  tilt?: number;
  style?: StyleProp<ViewStyle>;
}

/** A strip of gold washi tape. Journal photos only. Decoration: hidden from screen readers. */
export function WashiTape({ width = 86, height = 24, tilt = -8, style }: WashiTapeProps) {
  const { colors } = useTheme();
  return (
    <View
      importantForAccessibility="no-hide-descendants"
      accessibilityElementsHidden
      style={[
        {
          width,
          height,
          backgroundColor: colors.gold,
          opacity: 0.9,
          transform: [{ rotate: `${tilt}deg` }],
        },
        style,
      ]}
    />
  );
}
