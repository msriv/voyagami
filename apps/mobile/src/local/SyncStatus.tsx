import { Pressable, View } from "react-native";

import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Text } from "@/ui";

import { useSyncStatus, type SyncState } from "./syncStatus";

const dot: Record<SyncState, "matcha" | "gold" | "white" | "vermilion"> = {
  synced: "matcha",
  syncing: "gold",
  offline: "white",
  attention: "vermilion",
};

/**
 * Where the trip's changes stand: all saved, sending, no signal, or something the server refused.
 * The words carry the state; the dot is decoration. Changes are announced to screen readers.
 * `onDark` is for the trip header, which is dark.
 */
export function SyncStatus({
  onDark = false,
  onPress,
}: {
  onDark?: boolean;
  onPress?: () => void;
}) {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { state, pending, failed } = useSyncStatus();
  const label =
    state === "attention"
      ? t("sync.attention", { count: failed })
      : state === "offline"
        ? pending > 0
          ? t("sync.offlineSaved", { count: pending })
          : t("sync.offline")
        : state === "syncing"
          ? t("sync.syncing", { count: pending })
          : t("sync.synced");
  const color = onDark ? "white" : "ink";

  // Something is waiting or refused: the badge opens the list of changes. All saved: nothing to open.
  const opens = onPress && state !== "synced";
  const Wrapper = opens ? Pressable : View;
  const wrapperProps = opens
    ? {
        accessibilityRole: "button" as const,
        accessibilityHint: t("sync.opensList"),
        onPress,
        style: { gap: 4, minHeight: 44, justifyContent: "center" as const },
      }
    : { accessibilityRole: "text" as const, style: { gap: 4 } };

  return (
    <Wrapper
      accessible
      accessibilityLabel={label}
      accessibilityLiveRegion="polite"
      {...wrapperProps}
    >
      <View style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
        <View
          style={{
            width: 10,
            height: 10,
            borderRadius: 5,
            backgroundColor: colors[dot[state]],
          }}
        />
        <Text variant="caption" color={color} style={{ fontWeight: "700" }}>
          {label}
        </Text>
      </View>
      {state === "offline" ? (
        <Text variant="caption" color={color}>
          {t("sync.offlineHint")}
        </Text>
      ) : null}
    </Wrapper>
  );
}
