import { useRouter } from "expo-router";
import { getLocales } from "expo-localization";
import { useCallback, useEffect, useState } from "react";
import { ScrollView, Switch, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError } from "@voyagami/api-client";
import { formatDate } from "@voyagami/locale";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { CONSENTS, useConsent } from "@/privacy/Consent";
import { exportMyData } from "@/privacy/exportData";
import { useTheme } from "@/theme";
import { Button, Card, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

/** Data and privacy: what Voyagami may use, each with its own switch you can change any time. */
export default function PrivacyScreen() {
  const { t } = useTranslation();
  const { colors, hitArea } = useTheme();
  const router = useRouter();
  const { choices, set } = useConsent();
  const { api } = useAuth();
  const [deleteOn, setDeleteOn] = useState<string | undefined>();
  const [confirming, setConfirming] = useState(false);
  const [note, setNote] = useState<string | undefined>();

  useEffect(() => {
    void api
      .unwrap(api.GET("/v1/me/deletion"))
      .then((d) => setDeleteOn(d.scheduled ? d.deleteOn : undefined))
      .catch(() => undefined);
  }, [api]);

  const exportData = useCallback(async () => {
    setNote(t("privacy.exporting"));
    try {
      setNote((await exportMyData(api)) ? undefined : t("privacy.exportUnavailable"));
    } catch {
      setNote(t("privacy.exportError"));
    }
  }, [api, t]);

  const schedule = async () => {
    try {
      const d = await api.unwrap(api.DELETE("/v1/me"));
      setDeleteOn(d.deleteOn);
      setNote(undefined);
    } catch (e) {
      setNote(
        t(
          e instanceof ApiError && e.code === "subscription_active"
            ? "privacy.deleteSubscription"
            : "privacy.deleteError",
        ),
      );
    }
    setConfirming(false);
  };

  const keep = async () => {
    try {
      await api.unwrap(api.DELETE("/v1/me/deletion"));
      setDeleteOn(undefined);
    } catch {
      setNote(t("privacy.deleteError"));
    }
  };

  const day = deleteOn
    ? formatDate(new Date(deleteOn), getLocales()[0]?.languageTag ?? "en", { withYear: true })
    : "";

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 14 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("privacy.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("privacy.title")}
          </Text>
        </View>
        <Text color="muted">{t("privacy.intro")}</Text>
        {CONSENTS.map((kind) => (
          <Card key={kind}>
            <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
              <View style={{ flex: 1, gap: 4 }}>
                <Text variant="bodyStrong">{t(`privacy.items.${kind}.title`)}</Text>
                <Text variant="caption">{t(`privacy.items.${kind}.body`)}</Text>
              </View>
              <Switch
                accessibilityRole="switch"
                accessibilityLabel={t(`privacy.items.${kind}.title`)}
                accessibilityState={{ checked: choices[kind] }}
                value={choices[kind]}
                onValueChange={(on) => set(kind, on)}
                style={{ minHeight: hitArea.min }}
              />
            </View>
          </Card>
        ))}

        <Card style={{ gap: 8 }}>
          <Text variant="bodyStrong" heading>
            {t("privacy.exportTitle")}
          </Text>
          <Text variant="caption">{t("privacy.exportBody")}</Text>
          <Button
            label={t("privacy.exportButton")}
            variant="secondary"
            onPress={() => void exportData()}
          />
        </Card>

        <Card style={{ gap: 8 }}>
          <Text variant="bodyStrong" heading>
            {t("privacy.deleteTitle")}
          </Text>
          {deleteOn ? (
            <>
              <Text accessibilityLiveRegion="polite">
                {t("privacy.deleteScheduled", { date: day })}
              </Text>
              <Button label={t("privacy.deleteKeep")} onPress={() => void keep()} />
            </>
          ) : confirming ? (
            <>
              <Text accessibilityLiveRegion="polite">{t("privacy.deleteConfirm")}</Text>
              <Button label={t("privacy.deleteYes")} onPress={() => void schedule()} />
              <Button
                label={t("privacy.deleteNo")}
                variant="ghost"
                onPress={() => setConfirming(false)}
              />
            </>
          ) : (
            <>
              <Text variant="caption">{t("privacy.deleteBody")}</Text>
              <Button
                label={t("privacy.deleteButton")}
                variant="secondary"
                onPress={() => setConfirming(true)}
              />
            </>
          )}
        </Card>
        {note ? <Text accessibilityLiveRegion="polite">{note}</Text> : null}
      </ScrollView>
    </SafeAreaView>
  );
}
