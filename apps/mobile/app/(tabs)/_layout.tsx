import { Redirect, Tabs } from "expo-router";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";

export default function TabsLayout() {
  const { t } = useTranslation();
  const { colors, fontFamily } = useTheme();
  const { onboarded } = useAuth();

  // First launch goes through onboarding once; after that, straight to Home.
  if (!onboarded) return <Redirect href="/onboarding/region" />;

  return (
    <Tabs
      screenOptions={{
        sceneStyle: { backgroundColor: colors.washi },
        headerTitleStyle: { fontFamily: fontFamily.display, color: colors.ink },
        headerStyle: { backgroundColor: colors.washi },
        tabBarActiveTintColor: colors.indigo,
        tabBarInactiveTintColor: colors.muted,
        tabBarLabelStyle: { fontFamily: fontFamily.bodySemiBold },
        tabBarStyle: { backgroundColor: colors.white, borderTopColor: colors.line },
      }}
    >
      <Tabs.Screen
        name="index"
        options={{ title: t("tabs.home"), tabBarAccessibilityLabel: t("tabs.home") }}
      />
      <Tabs.Screen
        name="explore"
        options={{ title: t("tabs.explore"), tabBarAccessibilityLabel: t("tabs.explore") }}
      />
      <Tabs.Screen
        name="profile"
        options={{ title: t("tabs.profile"), tabBarAccessibilityLabel: t("tabs.profile") }}
      />
    </Tabs>
  );
}
