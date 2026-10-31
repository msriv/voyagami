import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { Linking, Platform, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { usePlanPrices } from "@/plans/prices";
import { useTheme } from "@/theme";
import { Button, Card, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

/** Where each store lets a person change or cancel a subscription. */
const STORE_PAGE = Platform.select({
  ios: "https://apps.apple.com/account/subscriptions",
  default: "https://play.google.com/store/account/subscriptions",
});

/** The plan you have, how to change or cancel it in plain words, and restore purchases. */
export default function ManageSubscriptionScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const prices = usePlanPrices();
  const [plan, setPlan] = useState<string | undefined>();
  const [notice, setNotice] = useState<string | undefined>();

  useEffect(() => {
    api
      .unwrap(api.GET("/v1/me/entitlements"))
      .then((e) => setPlan(e.plan))
      .catch(() => setPlan(undefined));
  }, [api]);

  const restore = () =>
    prices.restore
      ? void prices
          .restore()
          .then(() => setNotice(t("plans.restored")))
          .catch(() => setNotice(t("plans.restoreFailed")))
      : setNotice(t("plans.restoreUnavailable"));

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("plans.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("manageSub.title")}
          </Text>
        </View>

        <Card style={{ gap: 6 }}>
          <Text variant="caption">{t("manageSub.current")}</Text>
          <Text variant="bodyStrong" heading>
            {plan === "plus" ? t("plans.plus") : t("plans.free")}
          </Text>
          <Text>{plan === "plus" ? t("manageSub.plusNote") : t("manageSub.freeNote")}</Text>
        </Card>

        <Card style={{ gap: 8 }}>
          <Text variant="bodyStrong" heading>
            {t("manageSub.cancelTitle")}
          </Text>
          <Text>{t("manageSub.cancelBody")}</Text>
          <Text variant="caption">{t("manageSub.keepData")}</Text>
          <Button
            label={t("manageSub.openStore")}
            variant="secondary"
            onPress={() => void Linking.openURL(STORE_PAGE)}
          />
        </Card>

        <Button label={t("plans.restore")} variant="ghost" onPress={restore} />
        {notice ? <Text accessibilityLiveRegion="polite">{notice}</Text> : null}
      </ScrollView>
    </SafeAreaView>
  );
}
