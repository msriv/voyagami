import { useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { usePlanPrices } from "@/plans/prices";
import { useTheme } from "@/theme";
import { Button, Card, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Plan = "free" | "plus" | "pass";

const FREE = ["trips", "places", "bestOrder", "currencies"] as const;
const PLUS = ["offlineMaps", "bestOrder", "currencies", "import", "trips", "recap"] as const;
const TIERS = ["weekend", "week", "long"] as const;

/** Free, Plus and Trip Pass side by side, with the person's current plan marked. Prices come from the store. */
export default function PlansScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const prices = usePlanPrices();
  const [plan, setPlan] = useState<Plan | undefined>();
  const [notice, setNotice] = useState<string | undefined>();

  useEffect(() => {
    api
      .unwrap(api.GET("/v1/me/entitlements"))
      .then((e) => setPlan(e.plan))
      .catch(() => setPlan(undefined));
  }, [api]);

  const price = (p: string | undefined) => p ?? t("plans.priceInStore");
  const restore = () =>
    prices.restore
      ? void prices
          .restore()
          .then(() => setNotice(t("plans.restored")))
          .catch(() => setNotice(t("plans.restoreFailed")))
      : setNotice(t("plans.restoreUnavailable"));

  const current = (p: Plan) =>
    plan === p ? (
      <Text variant="caption" color="indigo" style={{ fontWeight: "800" }}>
        {t("plans.yourPlan")}
      </Text>
    ) : null;

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
            {t("plans.title")}
          </Text>
        </View>

        <Card style={{ gap: 6 }}>
          {current("free")}
          <Text variant="bodyStrong" heading>
            {t("plans.free")}
          </Text>
          <Text variant="caption">{t("plans.freeNote")}</Text>
          {FREE.map((k) => (
            <Text key={k}>{t(`plans.freeFeatures.${k}`)}</Text>
          ))}
        </Card>

        <Card style={{ gap: 6 }}>
          {current("plus")}
          <Text variant="bodyStrong" heading>
            {t("plans.plus")}
          </Text>
          <Text variant="title">{price(prices.plusYearly)}</Text>
          <Text variant="caption">{t("plans.plusYearly")}</Text>
          <Text variant="bodyStrong">{t("plans.trial")}</Text>
          {PLUS.map((k) => (
            <Text key={k}>{t(`plans.plusFeatures.${k}`)}</Text>
          ))}
          <Text variant="caption">{t("plans.crewNote")}</Text>
        </Card>

        <Card style={{ gap: 6 }}>
          {current("pass")}
          <Text variant="bodyStrong" heading>
            {t("plans.pass")}
          </Text>
          <Text variant="caption">{t("plans.passNote")}</Text>
          {TIERS.map((k) => (
            <View
              key={k}
              accessible
              accessibilityLabel={t("plans.tierLine", {
                name: t(`plans.tiers.${k}`),
                days: t(`plans.tierDays.${k}`),
                price: price(prices.pass?.[k]),
              })}
              style={{ flexDirection: "row", justifyContent: "space-between", gap: 12 }}
            >
              <Text style={{ flex: 1 }}>
                {t(`plans.tiers.${k}`)} <Text variant="caption">{t(`plans.tierDays.${k}`)}</Text>
              </Text>
              <Text variant="bodyStrong">{price(prices.pass?.[k])}</Text>
            </View>
          ))}
          <Text variant="caption">{t("plans.passCredit")}</Text>
        </Card>

        <Text variant="caption">{t("plans.alwaysFree")}</Text>
        <Button label={t("plans.restore")} variant="ghost" onPress={restore} />
        {notice ? <Text accessibilityLiveRegion="polite">{notice}</Text> : null}
      </ScrollView>
    </SafeAreaView>
  );
}
