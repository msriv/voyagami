import { useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Card, Chip, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

const KINDS = ["memberJoined", "stopAdded", "pollCreated", "expenseAdded", "meetPoint"] as const;

type Settings = Record<string, boolean>;

/** Push settings: one master switch and one per kind. A missing key means on. */
export default function NotificationSettingsScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const [settings, setSettings] = useState<Settings>({});
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    void api
      .unwrap(api.GET("/v1/me"))
      .then((me) => setSettings(me.notifications ?? {}))
      .catch(() => undefined);
  }, [api]);

  const isOn = (key: string) => settings[key] !== false;

  const toggle = useCallback(
    async (key: string) => {
      const before = settings;
      const next = { ...settings, [key]: !isOn(key) };
      setSettings(next);
      setError(undefined);
      try {
        await api.unwrap(api.PATCH("/v1/me", { body: { notifications: next } }));
      } catch (e) {
        setSettings(before);
        setError(
          t(
            e instanceof NetworkError
              ? "notificationSettings.offline"
              : "notificationSettings.saveFailed",
          ),
        );
      }
    },
    [api, settings, t],
  );

  const row = (key: string, label: string, disabled = false) => (
    <View key={key} style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
      <Text style={{ flex: 1 }}>{label}</Text>
      <Chip
        label={`${label}: ${t(isOn(key) ? "notificationSettings.on" : "notificationSettings.off")}`}
        selected={isOn(key)}
        disabled={disabled}
        onPress={() => void toggle(key)}
      />
    </View>
  );

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("notificationSettings.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("notificationSettings.title")}
          </Text>
        </View>
        <Text color="muted">{t("notificationSettings.intro")}</Text>
        <Card>
          <View style={{ gap: 14 }}>
            {row("all", t("notificationSettings.all"))}
            {KINDS.map((k) => row(k, t(`notificationSettings.kinds.${k}`), !isOn("all")))}
          </View>
        </Card>
        {error ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}
