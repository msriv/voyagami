import { useId } from "react";
import { StyleSheet } from "react-native";
import Svg, { Circle, Defs, G, Pattern, Rect } from "react-native-svg";

import { useTheme } from "@/theme";

export interface WavePatternProps {
  /** Line colour. Default white, for use on indigo. */
  color?: string;
  /** 0.09 to 0.12 behind hero cards, per the Brand board. Default 0.1. */
  opacity?: number;
}

const rings = [13, 8, 3];

/** Overlapping-wave pattern that fills its parent. Decoration only: hidden from screen readers. */
export function WavePattern({ color, opacity = 0.1 }: WavePatternProps) {
  const { colors } = useTheme();
  const stroke = color ?? colors.white;
  const id = useId().replace(/[^A-Za-z0-9]/g, "");
  return (
    <Svg
      pointerEvents="none"
      importantForAccessibility="no-hide-descendants"
      accessibilityElementsHidden
      style={[StyleSheet.absoluteFill, { opacity }]}
      width="100%"
      height="100%"
    >
      <Defs>
        <Pattern id={id} width={28} height={14} patternUnits="userSpaceOnUse">
          <G fill="none" stroke={stroke} strokeWidth={1}>
            {[
              [14, 14],
              [0, 7],
              [28, 7],
            ].flatMap(([cx, cy]) =>
              rings.map((r) => <Circle key={`${cx}-${cy}-${r}`} cx={cx} cy={cy} r={r} />),
            )}
          </G>
        </Pattern>
      </Defs>
      <Rect width="100%" height="100%" fill={`url(#${id})`} />
    </Svg>
  );
}
