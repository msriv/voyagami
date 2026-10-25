import { useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NetworkError } from "@voyagami/api-client";

import { useAuth, withDetail } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Chip, Text } from "@/ui";

const interests = [
  "mountains",
  "beaches",
  "foodTrails",
  "roadTrips",
  "cultureHistory",
  "nightlife",
  "wildlife",
  "slowCafes",
] as const;
const paces = ["slow", "balanced", "packed"] as const;
const spends = ["shoestring", "comfort", "splurge"] as const;

export default function StyleScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  const { api } = useAuth();
  const [picked, setPicked] = useState<string[]>([]);
  const [pace, setPace] = useState<(typeof paces)[number] | undefined>();
  const [spend, setSpend] = useState<(typeof spends)[number] | undefined>();
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const toggle = (key: string) =>
    setPicked((list) => (list.includes(key) ? list.filter((k) => k !== key) : [...list, key]));

  async function save() {
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.PATCH("/v1/me", {
          body: {
            travelStyle: {
              interests: picked as (typeof interests)[number][],
              ...(pace ? { pace } : null),
              ...(spend ? { spend } : null),
            },
          },
        }),
      );
      router.replace("/onboarding/connect");
    } catch (e) {
      setError(
        withDetail(
          t(e instanceof NetworkError ? "signIn.errors.offline" : "signIn.errors.generic"),
          e,
          true,
        ),
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 24, gap: 24 }}>
        <View style={{ gap: 8 }}>
          <Text variant="display" heading>
            {t("onboarding.style.title")}
          </Text>
          <Text color="muted">{t("onboarding.style.intro")}</Text>
        </View>
        <Group title={t("onboarding.style.interests")}>
          {interests.map((k) => (
            <Chip
              key={k}
              label={t(`onboarding.style.interest.${k}`)}
              selected={picked.includes(k)}
              onPress={() => toggle(k)}
            />
          ))}
        </Group>
        <Group title={t("onboarding.style.pace")}>
          {paces.map((k) => (
            <Chip
              key={k}
              label={t(`onboarding.style.paceOption.${k}`)}
              selected={pace === k}
              onPress={() => setPace(pace === k ? undefined : k)}
            />
          ))}
        </Group>
        <Group title={t("onboarding.style.spend")}>
          {spends.map((k) => (
            <Chip
              key={k}
              label={t(`onboarding.style.spendOption.${k}`)}
              selected={spend === k}
              onPress={() => setSpend(spend === k ? undefined : k)}
            />
          ))}
        </Group>
        {error ? (
          <Text color="vermilion" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        <Button label={t("onboarding.style.continue")} onPress={save} loading={busy} fullWidth />
        <Button
          label={t("onboarding.style.skip")}
          variant="ghost"
          onPress={() => router.replace("/onboarding/connect")}
          disabled={busy}
          fullWidth
        />
      </ScrollView>
    </SafeAreaView>
  );
}

function Group({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View style={{ gap: 12 }}>
      <Text variant="title" heading>
        {title}
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>{children}</View>
    </View>
  );
}
