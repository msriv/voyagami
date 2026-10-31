import { formatMoney } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { flushQueue } from "@/local/offlineHook";
import {
  discardPending,
  dismissFailed,
  failedWrites,
  pendingWrites,
  retryFailed,
  retryNow,
  type FailedEntry,
  type QueueEntry,
} from "@/local/queue";
import { onQueueChange } from "@/local/changes";
import { SyncStatus } from "@/local/SyncStatus";
import { useTheme } from "@/theme";
import { Button, Card, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

/** The changes this trip has not sent yet, and the ones the server refused, each with what to do about it. */
export default function SyncScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [waiting, setWaiting] = useState<QueueEntry[]>([]);
  const [refused, setRefused] = useState<FailedEntry[]>([]);

  const read = useCallback(async () => {
    const [p, f] = await Promise.all([pendingWrites(), failedWrites()]);
    setWaiting(p.filter((e) => e.tripId === id));
    setRefused(f.filter((e) => e.tripId === id));
  }, [id]);

  useEffect(() => {
    void read();
    return onQueueChange(() => void read());
  }, [read]);

  const describe = (e: QueueEntry) =>
    t("sync.expenseLine", {
      what: e.body.description,
      amount: formatMoney(e.body.original.amountMinor, e.body.original.currency, locale),
    });

  const sendNow = (entryId: string) => void retryNow(entryId).then(() => flushQueue(api));
  const tryAgain = (entryId: string) => void retryFailed(entryId).then(() => flushQueue(api));

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("sync.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("sync.title")}
          </Text>
        </View>
        <SyncStatus />

        {waiting.length === 0 && refused.length === 0 ? <Text>{t("sync.nothing")}</Text> : null}

        {refused.length > 0 ? (
          <View style={{ gap: 8 }}>
            <Text variant="bodyStrong" heading>
              {t("sync.refusedHeading")}
            </Text>
            <Text variant="caption">{t("sync.refusedExplain")}</Text>
            {refused.map((e) => (
              <Card key={e.id} style={{ gap: 8 }}>
                <Text variant="bodyStrong">{describe(e)}</Text>
                <Text>{t("sync.refusedReason", { reason: e.reason })}</Text>
                <View style={{ flexDirection: "row", gap: 8, flexWrap: "wrap" }}>
                  <Button
                    label={t("sync.tryAgain")}
                    accessibilityLabel={t("sync.tryAgainFor", { what: describe(e) })}
                    onPress={() => tryAgain(e.id)}
                  />
                  <Button
                    label={t("sync.discard")}
                    accessibilityLabel={t("sync.discardFor", { what: describe(e) })}
                    variant="secondary"
                    onPress={() => void dismissFailed(e.id)}
                  />
                </View>
              </Card>
            ))}
          </View>
        ) : null}

        {waiting.length > 0 ? (
          <View style={{ gap: 8 }}>
            <Text variant="bodyStrong" heading>
              {t("sync.waitingHeading")}
            </Text>
            {waiting.map((e) => (
              <Card key={e.id} style={{ gap: 8 }}>
                <Text variant="bodyStrong">{describe(e)}</Text>
                <View style={{ flexDirection: "row", gap: 8, flexWrap: "wrap" }}>
                  <Button
                    label={t("sync.sendNow")}
                    accessibilityLabel={t("sync.sendNowFor", { what: describe(e) })}
                    onPress={() => sendNow(e.id)}
                  />
                  <Button
                    label={t("sync.discard")}
                    accessibilityLabel={t("sync.discardFor", { what: describe(e) })}
                    variant="secondary"
                    onPress={() => void discardPending(e.id)}
                  />
                </View>
              </Card>
            ))}
          </View>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}
