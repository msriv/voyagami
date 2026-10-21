import { Tabs } from "expo-router";

// Tab titles are plain placeholders; VOY-99 moves them behind i18n and the theme.
export default function TabsLayout() {
  return (
    <Tabs>
      <Tabs.Screen name="index" options={{ title: "Home", tabBarAccessibilityLabel: "Home" }} />
      <Tabs.Screen
        name="explore"
        options={{ title: "Explore", tabBarAccessibilityLabel: "Explore" }}
      />
      <Tabs.Screen
        name="profile"
        options={{ title: "Profile", tabBarAccessibilityLabel: "Profile" }}
      />
    </Tabs>
  );
}
