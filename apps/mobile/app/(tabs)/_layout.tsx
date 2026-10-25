import type { ComponentProps } from "react";
import { Redirect, Tabs } from "expo-router";

import { useAuth } from "@/auth";
import { FloatingTabBar } from "@/ui/FloatingTabBar";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";

export default function TabsLayout() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { onboarded } = useAuth();

  // First launch goes through onboarding once; after that, straight to Home.
  if (!onboarded) return <Redirect href="/onboarding/region" />;

  return (
    <Tabs
      tabBar={(props) => <FloatingTabBar {...(props as unknown as ComponentProps<typeof FloatingTabBar>)} />}
      screenOptions={{ headerShown: false, sceneStyle: { backgroundColor: colors.washi } }}
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
