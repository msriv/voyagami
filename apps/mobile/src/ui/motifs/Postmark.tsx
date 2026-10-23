import { View } from "react-native";
import Svg, { Circle, Path } from "react-native-svg";

import { useTheme } from "@/theme";

import { Text } from "../Text";

export interface PostmarkProps {
  /** City name in plain English, from the destination. */
  city: string;
  /** Date, already formatted for the person's locale. */
  date: string;
}

/**
 * A postmark for journal days: double ring, wavy cancel lines, then city and date.
 * The text is real text so it scales and is read aloud as "city, date".
 */
export function Postmark({ city, date }: PostmarkProps) {
  const { colors } = useTheme();
  return (
    <View
      accessible
      accessibilityLabel={`${city}, ${date}`}
      style={{ flexDirection: "row", alignItems: "center", gap: 8, flexWrap: "wrap" }}
    >
      <Svg
        width={96}
        height={56}
        viewBox="0 0 96 56"
        importantForAccessibility="no-hide-descendants"
        accessibilityElementsHidden
      >
        <Circle cx={28} cy={28} r={22} fill="none" stroke={colors.indigo} strokeWidth={2.5} />
        <Circle cx={28} cy={28} r={15} fill="none" stroke={colors.indigo} strokeWidth={1.2} />
        <Path
          d="M50 18 q8 -5 16 0 t16 0 t14 0 M50 28 q8 -5 16 0 t16 0 t14 0 M50 38 q8 -5 16 0 t16 0 t14 0"
          fill="none"
          stroke={colors.indigo}
          strokeWidth={2}
        />
      </Svg>
      <View importantForAccessibility="no-hide-descendants" accessibilityElementsHidden>
        <Text variant="bodyStrong" color="indigo">
          {city}
        </Text>
        <Text variant="caption" color="indigo">
          {date}
        </Text>
      </View>
    </View>
  );
}
