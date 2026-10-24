import * as Contacts from "expo-contacts";
import * as Notifications from "expo-notifications";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Card, Text } from "@/ui";

type Ask = "contacts" | "notifications";

const requests: Record<Ask, () => Promise<{ granted: boolean }>> = {
  contacts: () => Contacts.requestPermissionsAsync(),
  notifications: () => Notifications.requestPermissionsAsync(),
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

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 24, gap: 24 }}>
        <View style={{ gap: 8 }}>
          <Text variant="display" heading>
            {t("onboarding.connect.title")}
          </Text>
          <Text color="muted">{t("onboarding.connect.intro")}</Text>
        </View>
        {(["contacts", "notifications"] as const).map((ask) => (
          <Card key={ask}>
            <View style={{ gap: 12 }}>
              <Text variant="title" heading>
                {t(`onboarding.connect.${ask}`)}
              </Text>
              <Text color="muted">{t(`onboarding.connect.${ask}Why`)}</Text>
              <Button
                label={
                  allowed[ask] ? t("onboarding.connect.allowed") : t("onboarding.connect.allow")
                }
                accessibilityLabel={`${t("onboarding.connect.allow")} ${t(`onboarding.connect.${ask}`)}`}
                variant="secondary"
                disabled={allowed[ask] === true}
                onPress={() => void allow(ask)}
              />
            </View>
          </Card>
        ))}
        <Button label={t("onboarding.connect.finish")} onPress={() => void finish()} fullWidth />
      </ScrollView>
    </SafeAreaView>
  );
}
