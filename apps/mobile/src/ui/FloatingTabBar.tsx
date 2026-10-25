import { Pressable, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { useTheme } from "@/theme";

import { Icon, type IconName } from "./icons";

const icons: Record<string, IconName> = { index: "home", explore: "compass", profile: "person" };

interface Route {
  key: string;
  name: string;
}
interface Props {
  state: { index: number; routes: Route[] };
  descriptors: Record<string, { options: { tabBarAccessibilityLabel?: string; title?: string } }>;
  navigation: {
    emit: (e: { type: "tabPress"; target: string; canPreventDefault: true }) => {
      defaultPrevented: boolean;
    };
    navigate: (name: string) => void;
  };
}

/** The floating dark pill from the Main board: 48px items, white square behind the selected one. */
export function FloatingTabBar({ state, descriptors, navigation }: Props) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <View
      style={{
        marginHorizontal: 16,
        marginBottom: Math.max(insets.bottom, 12) + 8,
        height: 68,
        borderRadius: 24,
        backgroundColor: colors.ink,
        flexDirection: "row",
        alignItems: "center",
        justifyContent: "space-around",
        paddingHorizontal: 12,
      }}
    >
      {state.routes.map((route, i) => {
        const selected = state.index === i;
        const { options } = descriptors[route.key] ?? { options: {} };
        return (
          <Pressable
            key={route.key}
            accessibilityRole="tab"
            accessibilityLabel={options.tabBarAccessibilityLabel ?? options.title}
            accessibilityState={{ selected }}
            onPress={() => {
              const event = navigation.emit({
                type: "tabPress",
                target: route.key,
                canPreventDefault: true,
              });
              if (!selected && !event.defaultPrevented) navigation.navigate(route.name);
            }}
            style={{
              width: 48,
              height: 48,
              borderRadius: 16,
              alignItems: "center",
              justifyContent: "center",
              backgroundColor: selected ? colors.white : "transparent",
            }}
          >
            <Icon name={icons[route.name] ?? "home"} color={selected ? colors.ink : "#C4C8D2"} />
          </Pressable>
        );
      })}
    </View>
  );
}
