import {
  defaultCurrency,
  defaultUnits,
  firstWeekday,
  regionName,
  searchRegions,
  unitOptions,
  type Units,
} from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { signInErrorKey, useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Card, Chip, Input, Sheet, Text } from "@/ui";

const maxShown = 40;

export default function RegionScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  const { api } = useAuth();
  const device = getLocales()[0];
  const tag = device?.languageTag ?? "en";
  const [region, setRegion] = useState<string | null>(device?.regionCode ?? null);
  const [currency, setCurrency] = useState(defaultCurrency(region) ?? "");
  const [units, setUnits] = useState<Units>(defaultUnits(region, tag));
  const [picking, setPicking] = useState(false);
  const [query, setQuery] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const results = searchRegions(query, tag);
  const currencyOk = /^[A-Z]{3}$/.test(currency);

  function choose(code: string) {
    setRegion(code);
    setCurrency(defaultCurrency(code) ?? "");
    setUnits(defaultUnits(code, tag));
    setPicking(false);
    setQuery("");
  }

  async function save() {
    if (!region) return;
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.PATCH("/v1/me", {
          body: {
            homeRegion: region,
            homeCurrency: currency,
            regionSource: "confirmed",
            units: { ...units, firstWeekday: firstWeekday(tag) },
          },
        }),
      );
      router.replace("/onboarding/style");
    } catch (e) {
      setError(t(`signIn.errors.${signInErrorKey(e)}`));
    } finally {
      setBusy(false);
    }
  }

  const choice = <K extends keyof Units>(key: K, options: readonly Units[K][], label: string) => (
    <View style={{ gap: 8 }}>
      <Text variant="bodyStrong">{label}</Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
        {options.map((value) => (
          <Chip
            key={value}
            label={t(`onboarding.region.unit.${value}`)}
            selected={units[key] === value}
            onPress={() => setUnits((u) => ({ ...u, [key]: value }))}
          />
        ))}
      </View>
    </View>
  );

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 24, gap: 24 }}>
        <View style={{ gap: 8 }}>
          <Text variant="display" heading>
            {t("onboarding.region.title")}
          </Text>
          <Text color="muted">
            {t(region ? "onboarding.region.intro" : "onboarding.region.introNone")}
          </Text>
        </View>
        <Card>
          <View style={{ gap: 12 }}>
            <Text variant="caption">{t("onboarding.region.homeRegion")}</Text>
            <Text variant="title" heading>
              {region ? regionName(region, tag) : t("onboarding.region.notSet")}
            </Text>
            <Button
              label={t(region ? "onboarding.region.change" : "onboarding.region.choose")}
              variant="secondary"
              onPress={() => setPicking(true)}
            />
          </View>
        </Card>
        {region ? (
          <>
            <Input
              label={t("onboarding.region.currency")}
              hint={t("onboarding.region.currencyHint")}
              value={currency}
              onChangeText={(text) => setCurrency(text.toUpperCase().replace(/[^A-Z]/g, ""))}
              autoCapitalize="characters"
              maxLength={3}
              error={currency && !currencyOk ? t("onboarding.region.currencyError") : undefined}
            />
            {choice("distance", unitOptions.distance, t("onboarding.region.distanceLabel"))}
            {choice(
              "temperature",
              unitOptions.temperature,
              t("onboarding.region.temperatureLabel"),
            )}
            {choice("clock", unitOptions.clock, t("onboarding.region.clockLabel"))}
          </>
        ) : null}
        {error ? (
          <Text color="vermilion" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        <Button
          label={t("onboarding.region.continue")}
          onPress={() => void save()}
          loading={busy}
          disabled={!region || !currencyOk}
          fullWidth
        />
      </ScrollView>
      <Sheet
        visible={picking}
        onClose={() => setPicking(false)}
        title={t("onboarding.region.pickTitle")}
        closeLabel={t("onboarding.region.close")}
        closeIcon={<Text>{"\u00d7"}</Text>}
      >
        <Input
          label={t("onboarding.region.search")}
          value={query}
          onChangeText={setQuery}
          autoCorrect={false}
        />
        {results.slice(0, maxShown).map((r) => (
          <Button
            key={r.code}
            label={r.name}
            variant={r.code === region ? "primary" : "secondary"}
            onPress={() => choose(r.code)}
          />
        ))}
        {results.length === 0 ? <Text color="muted">{t("onboarding.region.noMatch")}</Text> : null}
        {results.length > maxShown ? (
          <Text color="muted">{t("onboarding.region.keepTyping")}</Text>
        ) : null}
      </Sheet>
    </SafeAreaView>
  );
}
