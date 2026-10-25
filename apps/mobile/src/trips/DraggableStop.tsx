import { useRef, type ReactNode } from "react";
import { Animated, PanResponder, View } from "react-native";

import { useTheme } from "@/theme";
import { Text } from "@/ui";

interface Props {
  /** False for people who can't edit: just the card, no handle. */
  enabled: boolean;
  label: string;
  /** Called on release with how many rows the card moved: negative is up. */
  onDrop: (rows: number) => void;
  children: ReactNode;
}

/** A card with a handle on the right. Drag the handle up or down; the card follows the finger. */
export function DraggableStop({ enabled, label, onDrop, children }: Props) {
  const { hitArea } = useTheme();
  const dy = useRef(new Animated.Value(0)).current;
  const height = useRef(0);
  const drop = useRef(onDrop);
  drop.current = onDrop;

  const pan = useRef(
    PanResponder.create({
      onStartShouldSetPanResponder: () => true,
      onPanResponderMove: Animated.event([null, { dy }], { useNativeDriver: false }),
      onPanResponderRelease: (_e, g) => {
        dy.setValue(0);
        // shortcut: rows are assumed as tall as this card, until stops get a fixed height.
        if (height.current > 0) drop.current(Math.round(g.dy / height.current));
      },
      onPanResponderTerminate: () => dy.setValue(0),
    }),
  ).current;

  if (!enabled) return <>{children}</>;

  return (
    <Animated.View
      onLayout={(e) => (height.current = e.nativeEvent.layout.height)}
      style={{ flexDirection: "row", alignItems: "center", transform: [{ translateY: dy }] }}
    >
      <View style={{ flex: 1 }}>{children}</View>
      <View
        {...pan.panHandlers}
        accessible
        accessibilityRole="adjustable"
        accessibilityLabel={label}
        accessibilityActions={[{ name: "increment" }, { name: "decrement" }]}
        onAccessibilityAction={(e) =>
          drop.current(e.nativeEvent.actionName === "increment" ? 1 : -1)
        }
        style={{
          width: hitArea.min,
          minHeight: hitArea.min,
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <Text variant="bodyStrong">{"≡"}</Text>
      </View>
    </Animated.View>
  );
}
