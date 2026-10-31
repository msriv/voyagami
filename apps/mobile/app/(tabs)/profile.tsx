import { regionName } from "@voyagami/locale";
import { useRouter } from "expo-router";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { useTheme } from "@/theme";
import { Button, Card, Text } from "@/ui";

export default function ProfileScreen() {
  const { t } = useTranslation();
  const { colors, radius } = useTheme();
  const { signOut, resetDevice } = useAuth();
  const { region, currency, locale, units, displayName } = useProfile();
  const router = useRouter();
  const name = displayName || t("profile.traveller");
  const unitLine = [
    t(`profile.unit_${units.distance}`),
    t(`profile.unit_${units.temperature}`),
    t(`profile.unit_${units.clock}`),
  ].join(" · ");
  return (
    <SafeAreaView
      style={{ flex: 1, backgroundColor: colors.washi }}
      accessibilityLabel={t("tabs.profile")}
    >
      <ScrollView contentContainerStyle={{ padding: 20, gap: 18 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 14 }}>
          <View
            accessibilityElementsHidden
            importantForAccessibility="no-hide-descendants"
            style={{
              width: 56,
              height: 56,
              borderRadius: radius.pill,
              backgroundColor: colors.indigo,
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Text variant="title" color="white">
              {name.slice(0, 1).toUpperCase()}
            </Text>
          </View>
          <Text variant="title" heading style={{ flexShrink: 1 }}>
            {name}
          </Text>
        </View>

        <Text variant="caption" heading>
          {t("profile.sectionHome")}
        </Text>
        <Card>
          <View style={{ gap: 12 }}>
            <View style={{ gap: 2 }}>
              <Text variant="caption">{t("profile.homeRegion")}</Text>
              <Text variant="bodyStrong">
                {region ? regionName(region, locale) : t("onboarding.region.notSet")}
                {currency ? ` · ${currency}` : ""}
              </Text>
            </View>
            <View style={{ gap: 2 }}>
              <Text variant="caption">{t("profile.units")}</Text>
              <Text variant="body">{unitLine}</Text>
            </View>
            <Button
              label={t("onboarding.region.change")}
              variant="secondary"
              onPress={() => router.push("/onboarding/region")}
            />
          </View>
        </Card>

        <Button
          label={t("profile.notifications")}
          variant="secondary"
          onPress={() => router.push("/notifications")}
        />

        <Text variant="caption" heading>
          {t("profile.sectionData")}
        </Text>
        <Card>
          <View style={{ gap: 4 }}>
            <Text variant="bodyStrong">{t("profile.dataRegion")}</Text>
            <Text variant="caption">{t("profile.dataNote")}</Text>
          </View>
        </Card>

        <Button label={t("profile.signOut")} variant="secondary" onPress={() => void signOut()} />
        <Button label={t("profile.resetApp")} variant="ghost" onPress={() => void resetDevice()} />
      </ScrollView>
    </SafeAreaView>
  );
}
