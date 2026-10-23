import Svg, { G, Path, Rect } from "react-native-svg";

import { useTheme } from "@/theme";

export interface SealProps {
  /** Width and height. The smallest seal is 24. */
  size?: number;
  /** Screen reader label. Omit when the seal is decoration. */
  accessibilityLabel?: string;
}

/** The paper-plane seal (design/brand/voyagami-seal.svg): logo, stamps and achievements. */
export function Seal({ size = 60, accessibilityLabel }: SealProps) {
  const { colors } = useTheme();
  const labelled = accessibilityLabel !== undefined;
  return (
    <Svg
      width={Math.max(size, 24)}
      height={Math.max(size, 24)}
      viewBox="0 0 1024 1024"
      importantForAccessibility={labelled ? "yes" : "no-hide-descendants"}
      {...(labelled ? { accessible: true, accessibilityRole: "image", accessibilityLabel } : null)}
    >
      <G rotation={-4} origin="512, 512">
        <Rect x={64} y={64} width={896} height={896} rx={200} fill={colors.vermilion} />
        <Path d="M806 236 L228 508 L446 588 Z" fill={colors.washi} />
        <Path d="M806 236 L446 588 L566 812 Z" fill={colors.sealFold1} />
        <Path d="M446 588 L566 812 L402 716 Z" fill={colors.sealFold2} />
        <Path
          d="M170 790 C 250 700, 330 700, 392 742"
          fill="none"
          stroke={colors.gold}
          strokeWidth={18}
          strokeLinecap="round"
          strokeDasharray="0.1 44"
        />
      </G>
    </Svg>
  );
}
