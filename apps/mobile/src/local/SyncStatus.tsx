import { View } from "react-native";

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
export function SyncStatus({ onDark = false }: { onDark?: boolean }) {
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

  return (
    <View
      accessible
      accessibilityRole="text"
      accessibilityLabel={label}
      accessibilityLiveRegion="polite"
      style={{ gap: 4 }}
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
    </View>
  );
}
