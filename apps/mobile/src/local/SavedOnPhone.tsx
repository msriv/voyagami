import { formatBytes } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useCallback, useEffect, useState } from "react";
import { View } from "react-native";

import { useTranslation } from "@/i18n";
import { Button, Text } from "@/ui";

import { failedWrites, pendingWrites } from "./queue";
import { localStore } from "./store";

/** Empties the saved copies of trips, but never changes waiting to send: those exist nowhere else. */
export async function clearSavedCopies() {
  const [waiting, refused] = await Promise.all([pendingWrites(), failedWrites()]);
  await localStore().clear();
  if (waiting.length > 0) await localStore().put("queue", waiting);
  if (refused.length > 0) await localStore().put("queue-failed", refused);
}

/** How much the phone keeps for offline use, with a way to clear it. Always free. */
export function SavedOnPhone() {
  const { t } = useTranslation();
  const [bytes, setBytes] = useState<number | undefined>();
  const locale = getLocales()[0]?.languageTag ?? "en";

  const read = useCallback(() => void localStore().sizeBytes().then(setBytes), []);
  useEffect(read, [read]);

  return (
    <View style={{ gap: 8 }}>
      <Text variant="bodyStrong">{t("storage.title")}</Text>
      <Text variant="caption" accessibilityLiveRegion="polite">
        {bytes === undefined
          ? t("storage.loading")
          : t("storage.used", { size: formatBytes(bytes, locale) })}
      </Text>
      <Text variant="caption">{t("storage.note")}</Text>
      <Button
        label={t("storage.clear")}
        variant="secondary"
        onPress={() => void clearSavedCopies().then(read)}
      />
    </View>
  );
}
