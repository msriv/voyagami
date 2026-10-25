import { useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Text } from "@/ui";

type Ask = "contacts" | "location" | "notifications";

// Loaded on tap, not at start-up: a native module that Expo Go lacks must not crash this screen.
const requests: Record<Ask, () => Promise<{ granted: boolean }>> = {
  /* eslint-disable @typescript-eslint/no-require-imports -- loaded on tap, see above */
  contacts: async () => require("expo-contacts").requestPermissionsAsync(),
  location: async () => require("expo-location").requestForegroundPermissionsAsync(),
  notifications: async () => require("expo-notifications").requestPermissionsAsync(),
  /* eslint-enable @typescript-eslint/no-require-imports */
};

export default function ConnectScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  const { finishOnboarding } = useAuth();
  const [allowed, setAllowed] = useState<Partial<Record<Ask, boolean>>>({});

  async function allow(ask: Ask) {
    // A denied or failed ask is fine: every one is optional.
    const result = await requests[ask]().catch(() => ({ granted: false }));
    setAllowed((a) => ({ ...a, [ask]: result.granted }));
  }

  async function finish() {
    await finishOnboarding();
    router.replace("/");
  }

  const tiles: Record<Ask, string> = {
    contacts: colors.gold,
    location: colors.matcha,
    notifications: colors.chipOff,
  };

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 24, gap: 16, flexGrow: 1 }}>
        <View
          style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}
        >
          <View
            accessible
            accessibilityLabel={t("onboarding.step", { n: 2, total: 2 })}
            style={{ flexDirection: "row", gap: 6 }}
          >
            {[0, 1].map((i) => (
              <View
                key={i}
                style={{ width: 32, height: 6, borderRadius: 99, backgroundColor: colors.ink }}
              />
            ))}
          </View>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("onboarding.connect.later")}
            onPress={() => void finish()}
            style={{ minHeight: 44, justifyContent: "center" }}
          >
            <Text variant="bodyStrong">{t("onboarding.connect.later")}</Text>
          </Pressable>
        </View>
        <View style={{ gap: 8 }}>
          <Text variant="display" heading>
            {t("onboarding.connect.title")}
          </Text>
          <Text color="muted">{t("onboarding.connect.intro")}</Text>
        </View>
        <View
          style={{
            backgroundColor: colors.white,
            borderRadius: 22,
            paddingHorizontal: 16,
            paddingVertical: 4,
          }}
        >
          {(["contacts", "location", "notifications"] as const).map((ask, i) => (
            <View
              key={ask}
              style={{
                flexDirection: "row",
                alignItems: "center",
                gap: 12,
                paddingVertical: 12,
                borderTopWidth: i === 0 ? 0 : 1,
                borderTopColor: colors.line,
              }}
            >
              <View
                style={{ width: 40, height: 40, borderRadius: 12, backgroundColor: tiles[ask] }}
              />
              <View style={{ flex: 1 }}>
                <Text variant="bodyStrong">{t(`onboarding.connect.${ask}`)}</Text>
                <Text variant="caption" color="muted">
                  {t(`onboarding.connect.${ask}Why`)}
                </Text>
              </View>
              <Button
                label={
                  allowed[ask] ? t("onboarding.connect.allowed") : t("onboarding.connect.allow")
                }
                accessibilityLabel={`${t("onboarding.connect.allow")} ${t(`onboarding.connect.${ask}`)}`}
                disabled={allowed[ask] === true}
                onPress={() => void allow(ask)}
              />
            </View>
          ))}
        </View>
        <View
          style={{
            borderWidth: 2,
            borderStyle: "dashed",
            borderColor: colors.line,
            borderRadius: 18,
            padding: 16,
            gap: 6,
          }}
        >
          <Text variant="bodyStrong">{t("onboarding.connect.autoTitle")}</Text>
          <Text variant="caption" color="muted">
            {t("onboarding.connect.autoBody")}
          </Text>
        </View>
        <View style={{ flex: 1 }} />
        <Button label={t("onboarding.connect.finish")} onPress={() => void finish()} fullWidth />
      </ScrollView>
    </SafeAreaView>
  );
}
