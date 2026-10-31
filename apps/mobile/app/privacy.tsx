import { useRouter } from "expo-router";
import { ScrollView, Switch, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useTranslation } from "@/i18n";
import { CONSENTS, useConsent } from "@/privacy/Consent";
import { useTheme } from "@/theme";
import { Card, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

/** Data and privacy: what Voyagami may use, each with its own switch you can change any time. */
export default function PrivacyScreen() {
  const { t } = useTranslation();
  const { colors, hitArea } = useTheme();
  const router = useRouter();
  const { choices, set } = useConsent();

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
      </ScrollView>
    </SafeAreaView>
  );
}
