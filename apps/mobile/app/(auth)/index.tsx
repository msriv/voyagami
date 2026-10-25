import { useRouter } from "expo-router";
import { View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Seal, Text, WavePattern } from "@/ui";

export default function WelcomeScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.indigo }}>
      <WavePattern opacity={0.16} />
      <View style={{ flex: 1, padding: 24, gap: 24, justifyContent: "flex-end" }}>
        <Seal size={96} />
        <Text variant="display" heading color="white">
          {t("onboarding.welcome.title")}
        </Text>
        <Text color="white">{t("onboarding.welcome.intro")}</Text>
        <Button
          label={t("onboarding.welcome.start")}
          variant="accent"
          onPress={() => router.push("/sign-in")}
          fullWidth
        />
      </View>
    </SafeAreaView>
  );
}
