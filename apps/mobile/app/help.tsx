import Constants from "expo-constants";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Linking, Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { diagnostics } from "@/help/diagnostics";
import { HELP_ARTICLES, type HelpArticleId } from "@/help/articles";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Card, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

const supportEmail = (): string | undefined =>
  (Constants.expoConfig?.extra?.supportEmail as string | undefined) ||
  process.env.EXPO_PUBLIC_SUPPORT_EMAIL ||
  undefined;

/** Help centre: short answers that open in place, and a way to reach us. */
export default function HelpScreen() {
  const { t } = useTranslation();
  const { colors, hitArea } = useTheme();
  const router = useRouter();
  const [open, setOpen] = useState<HelpArticleId | undefined>();
  const email = supportEmail();

  const contact = () =>
    void Linking.openURL(
      `mailto:${email}?subject=${encodeURIComponent(t("help.contactSubject"))}&body=${encodeURIComponent(`\n\n${diagnostics()}`)}`,
    );

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 12 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("help.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("help.title")}
          </Text>
        </View>
        <Text color="muted">{t("help.intro")}</Text>

        {HELP_ARTICLES.map((id) => {
          const expanded = open === id;
          return (
            <Card key={id} style={{ gap: 8 }}>
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={t(`help.articles.${id}.title`)}
                accessibilityState={{ expanded }}
                onPress={() => setOpen(expanded ? undefined : id)}
                style={{ minHeight: hitArea.min, justifyContent: "center" }}
              >
                <Text variant="bodyStrong">{t(`help.articles.${id}.title`)}</Text>
              </Pressable>
              {expanded ? <Text>{t(`help.articles.${id}.body`)}</Text> : null}
            </Card>
          );
        })}

        <Card style={{ gap: 8 }}>
          <Text variant="caption" color="muted">
            {t("help.contactNote")}
          </Text>
          {email ? (
            <Button label={t("help.contact")} onPress={contact} />
          ) : (
            <Text accessibilityLiveRegion="polite">{t("help.contactUnavailable")}</Text>
          )}
        </Card>
      </ScrollView>
    </SafeAreaView>
  );
}
