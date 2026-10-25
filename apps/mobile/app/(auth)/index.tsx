import { useRouter } from "expo-router";
import { View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Seal, Text, WavePattern } from "@/ui";

// Welcome board: three tilted cards, a stamp, then the promise and two ways in.
export default function WelcomeScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();

  const card = (
    key: "plan" | "live" | "keep",
    background: string,
    left: `${number}%`,
    top: number,
    rotate: string,
    shadow: boolean,
  ) => (
    <View
      key={key}
      style={{
        position: "absolute",
        left,
        top,
        width: "44%",
        minHeight: 186,
        backgroundColor: background,
        borderRadius: 18,
        padding: 16,
        justifyContent: "space-between",
        transform: [{ rotate }],
        ...(shadow
          ? {
              shadowColor: "#0A1450",
              shadowOpacity: 0.3,
              shadowRadius: 18,
              shadowOffset: { width: 0, height: 12 },
              elevation: 6,
            }
          : null),
      }}
    >
      <Text
        variant="caption"
        style={{
          color: key === "keep" ? colors.indigo : colors.ink,
          textTransform: "uppercase",
          letterSpacing: 1,
        }}
      >
        {t(`onboarding.welcome.cards.${key}.label`)}
      </Text>
      <Text variant="display" style={{ fontSize: 24, lineHeight: 26 }}>
        {t(`onboarding.welcome.cards.${key}.text`)}
      </Text>
    </View>
  );

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.indigo }}>
      <WavePattern opacity={0.09} />
      <View style={{ flex: 1, padding: 24, gap: 12 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
          <Text variant="accent" color="white" style={{ fontSize: 32 }}>
            {t("home.brand")}
          </Text>
          <Seal size={24} />
        </View>

        <View style={{ height: 330, marginTop: 12 }} accessibilityElementsHidden>
          {card("plan", colors.gold, "3%", 40, "-9deg", false)}
          {card("live", colors.vermilion, "34%", 8, "4deg", true)}
          {card("keep", colors.white, "54%", 128, "-3deg", true)}
          <View
            style={{
              position: "absolute",
              left: "10%",
              top: 238,
              width: 76,
              height: 76,
              borderRadius: 38,
              borderWidth: 3,
              borderColor: colors.white,
              transform: [{ rotate: "-14deg" }],
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Text
              variant="caption"
              color="white"
              style={{ fontSize: 9, textTransform: "uppercase", letterSpacing: 1 }}
            >
              {t("onboarding.welcome.stamp")}
            </Text>
          </View>
        </View>

        <View style={{ flex: 1 }} />
        <Text variant="display" heading color="white" style={{ fontSize: 40, lineHeight: 42 }}>
          {t("onboarding.welcome.titleStart")}
          <Text variant="accent" color="white" style={{ fontSize: 40, lineHeight: 42 }}>
            {t("onboarding.welcome.titleAccent")}
          </Text>
        </Text>
        <Text color="white">{t("onboarding.welcome.intro")}</Text>
        <View style={{ gap: 10, marginTop: 12 }}>
          <Button
            label={t("onboarding.welcome.start")}
            variant="accent"
            onPress={() => router.push("/sign-in")}
            fullWidth
          />
          <Button
            label={t("onboarding.welcome.invite")}
            variant="secondary"
            onPress={() => router.push("/sign-in")}
            fullWidth
          />
        </View>
      </View>
    </SafeAreaView>
  );
}
