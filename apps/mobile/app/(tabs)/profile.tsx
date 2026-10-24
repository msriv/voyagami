import { regionName } from "@voyagami/locale";
import { useRouter } from "expo-router";
import { View } from "react-native";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { Button, Card, Text } from "@/ui";

export default function ProfileScreen() {
  const { t } = useTranslation();
  const { signOut } = useAuth();
  const { region, currency, locale } = useProfile();
  const router = useRouter();
  return (
    <View style={{ flex: 1, padding: 24, gap: 16 }} accessibilityLabel={t("tabs.profile")}>
      <Card>
        <View style={{ gap: 12 }}>
          <Text variant="caption">{t("profile.homeRegion")}</Text>
          <Text variant="title" heading>
            {region ? regionName(region, locale) : t("onboarding.region.notSet")}
            {currency ? ` · ${currency}` : ""}
          </Text>
          <Button
            label={t("onboarding.region.change")}
            variant="secondary"
            onPress={() => router.push("/onboarding/region")}
          />
        </View>
      </Card>
      <Button label={t("profile.signOut")} variant="secondary" onPress={() => void signOut()} />
    </View>
  );
}
