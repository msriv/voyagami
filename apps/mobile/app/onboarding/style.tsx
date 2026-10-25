import { useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
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
        <View
          style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}
        >
          <View
            accessible
            accessibilityLabel={t("onboarding.step", { n: 1, total: 2 })}
            style={{ flexDirection: "row", gap: 6 }}
          >
            {[colors.ink, colors.chipOff].map((c, i) => (
              <View
                key={i}
                style={{ width: 32, height: 6, borderRadius: 99, backgroundColor: c }}
              />
            ))}
          </View>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("onboarding.style.skip")}
            disabled={busy}
            onPress={() => router.replace("/onboarding/connect")}
            style={{ minHeight: 44, justifyContent: "center" }}
          >
            <Text variant="bodyStrong">{t("onboarding.style.skipShort")}</Text>
          </Pressable>
        </View>
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
        <Segmented
          title={t("onboarding.style.pace")}
          options={paces.map((k) => ({ key: k, label: t(`onboarding.style.paceOption.${k}`) }))}
          value={pace}
          onChange={setPace}
        />
        <Segmented
          title={t("onboarding.style.spend")}
          options={spends.map((k) => ({ key: k, label: t(`onboarding.style.spendOption.${k}`) }))}
          value={spend}
          onChange={setSpend}
        />
        {error ? (
          <Text color="vermilion" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        <Button label={t("onboarding.style.continue")} onPress={save} loading={busy} fullWidth />
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

function Segmented<K extends string>({
  title,
  options,
  value,
  onChange,
}: {
  title: string;
  options: { key: K; label: string }[];
  value: K | undefined;
  onChange: (key: K | undefined) => void;
}) {
  const { colors } = useTheme();
  return (
    <View style={{ gap: 8 }}>
      <Text variant="bodyStrong">{title}</Text>
      <View
        style={{
          flexDirection: "row",
          gap: 4,
          backgroundColor: colors.white,
          borderRadius: 16,
          padding: 4,
        }}
      >
        {options.map((o) => (
          <Pressable
            key={o.key}
            accessibilityRole="button"
            accessibilityState={{ selected: value === o.key }}
            onPress={() => onChange(value === o.key ? undefined : o.key)}
            style={{
              flex: 1,
              minHeight: 44,
              borderRadius: 12,
              alignItems: "center",
              justifyContent: "center",
              backgroundColor: value === o.key ? colors.vermilion : "transparent",
            }}
          >
            <Text variant={value === o.key ? "bodyStrong" : "body"}>{o.label}</Text>
          </Pressable>
        ))}
      </View>
    </View>
  );
}
