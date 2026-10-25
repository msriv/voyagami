import type { ErrorBoundaryProps } from "expo-router";
import { useEffect } from "react";
import { Pressable, ScrollView, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { clearDevice } from "@/auth";
import { i18n } from "@/i18n";
import { reportError } from "@/logging/remoteLog";
import { colors } from "@/theme";

/**
 * Shown when a screen fails to render. It sits above the providers, so it uses plain views and
 * the raw tokens: the message is on screen, and Reset wipes this device back to Welcome.
 */
export function ErrorScreen({ error, retry }: ErrorBoundaryProps) {
  useEffect(() => reportError("error screen", error), [error]);
  const button = (label: string, onPress: () => void, dark: boolean) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={{
        minHeight: 48,
        borderRadius: 16,
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: dark ? colors.ink : colors.white,
      }}
    >
      <Text style={{ fontWeight: "800", color: dark ? colors.white : colors.ink }}>{label}</Text>
    </Pressable>
  );
  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 24, gap: 16 }}>
        <Text accessibilityRole="header" style={{ fontSize: 28, fontWeight: "800" }}>
          {i18n.t("errorScreen.title")}
        </Text>
        <Text selectable style={{ color: colors.muted }}>
          {`${error.name}: ${error.message}`}
        </Text>
        <View style={{ gap: 8 }}>
          {button(i18n.t("errorScreen.retry"), () => void retry(), true)}
          {button(
            i18n.t("errorScreen.reset"),
            () => void clearDevice().finally(() => void retry()),
            false,
          )}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
