import Svg, { Circle, Path } from "react-native-svg";

export type IconName = "home" | "compass" | "book" | "people" | "person";

// Stroke icons on a 24x24 grid, paths from design/boards/Main.dc.html. Bundled so no icon font is needed.
export function Icon({ name, size = 24, color }: { name: IconName; size?: number; color: string }) {
  return (
    <Svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      importantForAccessibility="no-hide-descendants"
    >
      {name === "home" ? <Path d="M3 10.5L12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z" /> : null}
      {name === "compass" ? (
        <>
          <Circle cx={12} cy={12} r={9} />
          <Path d="M15.5 8.5l-2 5-5 2 2-5z" />
        </>
      ) : null}
      {name === "book" ? (
        <>
          <Path d="M5 4h11a3 3 0 0 1 3 3v13H8a3 3 0 0 1-3-3z" />
          <Path d="M5 17a3 3 0 0 1 3-3h11" />
        </>
      ) : null}
      {name === "people" || name === "person" ? (
        <>
          <Circle cx={name === "person" ? 12 : 9} cy={8} r={4} />
          <Path
            d={
              name === "person"
                ? "M5 21v-1a7 7 0 0 1 14 0v1"
                : "M2 21v-1a6 6 0 0 1 12 0v1M16 4a4 4 0 0 1 0 8M22 21v-1a6 6 0 0 0-3-5"
            }
          />
        </>
      ) : null}
    </Svg>
  );
}
