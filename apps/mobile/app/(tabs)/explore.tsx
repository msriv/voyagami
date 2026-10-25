import { regionName } from "@voyagami/locale";
import { useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { useTheme } from "@/theme";
import { suggestedDestinations } from "@/trips/destinations";
import { Text } from "@/ui";

export default function ExploreScreen() {
  const { t } = useTranslation();
  const { colors, radius, hitArea } = useTheme();
  const { locale } = useProfile();
  const router = useRouter();
  const [query, setQuery] = useState("");

  const q = query.trim().toLowerCase();
  const places = suggestedDestinations
    .map((d) => ({ ...d, country: regionName(d.countryCode, locale) }))
    .filter((d) => !q || `${d.name} ${d.country}`.toLowerCase().includes(q));

  return (
    <SafeAreaView
      style={{ flex: 1, backgroundColor: colors.washi }}
      accessibilityLabel={t("tabs.explore")}
    >
      <ScrollView
        contentContainerStyle={{ padding: 20, gap: 16 }}
        keyboardShouldPersistTaps="handled"
      >
        <Text variant="display" heading>
          {t("explore.title")}
        </Text>
        <TextInput
          accessibilityLabel={t("explore.searchLabel")}
          placeholder={t("explore.searchPlaceholder")}
          placeholderTextColor={colors.muted}
          value={query}
          onChangeText={setQuery}
          returnKeyType="search"
          autoCorrect={false}
          style={{
            minHeight: hitArea.min,
            paddingHorizontal: 16,
            borderRadius: radius.pill,
            borderWidth: 1,
            borderColor: colors.line,
            backgroundColor: colors.white,
            color: colors.ink,
            fontSize: 16,
          }}
        />
        <View style={{ gap: 2 }}>
          <Text variant="title" heading>
            {t("explore.sectionTitle")}
          </Text>
          <Text variant="caption">{t("explore.sectionBody")}</Text>
        </View>
        {places.length === 0 ? <Text color="muted">{t("explore.empty")}</Text> : null}
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 12 }}>
          {places.map((d) => (
            <Pressable
              key={d.name}
              accessibilityRole="button"
              accessibilityLabel={t("explore.plan", { place: d.name })}
              hitSlop={4}
              onPress={() =>
                router.push({ pathname: "/trips/new", params: { destination: d.name } })
              }
              style={({ pressed }) => ({
                flexGrow: 1,
                flexBasis: "45%",
                minHeight: 96,
                justifyContent: "flex-end",
                padding: 14,
                borderRadius: radius.card,
                backgroundColor: colors.matcha,
                opacity: pressed ? 0.85 : 1,
              })}
            >
              <Text variant="title">{d.name}</Text>
              <Text variant="caption" color="ink">
                {d.country}
              </Text>
            </Pressable>
          ))}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
