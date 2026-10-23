import Svg, { Path } from "react-native-svg";

import { useTheme } from "@/theme";

export interface BrushCircleProps {
  size?: number;
  color?: string;
}

/** A hand-brushed circle with an open end. Used at trip start and in the recap. Decoration only. */
export function BrushCircle({ size = 60, color }: BrushCircleProps) {
  const { colors } = useTheme();
  return (
    <Svg
      width={size}
      height={size}
      viewBox="0 0 88 88"
      importantForAccessibility="no-hide-descendants"
      accessibilityElementsHidden
    >
      <Path
        d="M70 22 A 34 34 0 1 0 76 52"
        fill="none"
        stroke={color ?? colors.ink}
        strokeWidth={7}
        strokeLinecap="round"
      />
    </Svg>
  );
}
