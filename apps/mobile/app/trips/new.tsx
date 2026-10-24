import { defaultCurrency } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useRouter } from "expo-router";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { signInErrorKey, useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { suggestedDestinations, type Destination } from "@/trips/destinations";
import { DateRangePicker } from "@/trips/DateRangePicker";
import type { DateRange } from "@/trips/dates";
import { Button, Chip, IconButton, Input, Text } from "@/ui";

type Mode = "dates" | "flexible";

export default function NewTripScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  const { api } = useAuth();
  const locale = getLocales()[0];
  const tag = locale?.languageTag ?? "en";

  const [stops, setStops] = useState<Destination[]>([]);
  const [mode, setMode] = useState<Mode>("dates");
  const [range, setRange] = useState<DateRange>({});
  const [days, setDays] = useState(7);
  const [name, setName] = useState("");
  const [currency, setCurrency] = useState(defaultCurrency(locale?.regionCode ?? null) ?? "");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const first = stops[0];

  async function save() {
    const code = currency.trim().toUpperCase();
    const dates =
      mode === "flexible"
        ? { flexible: { days } }
        : range.start && range.end
          ? { start: range.start, end: range.end }
          : undefined;
    const problem = !first
      ? t("newTrip.errors.noStops")
      : !dates
        ? t("newTrip.errors.noDates")
        : !/^[A-Z]{3}$/.test(code)
          ? t("newTrip.errors.badCurrency")
          : undefined;
    setError(problem);
    if (problem || !first || !dates) return;
    setBusy(true);
    try {
      await api.unwrap(
        api.POST("/v1/trips", {
          body: {
            title: name.trim() || t("newTrip.defaultName", { place: first.name }),
            destinations: stops,
            dates,
            timeZone: first.timeZone,
            settlementCurrency: code,
          },
        }),
      );
      router.replace("/");
    } catch (e) {
      const key = signInErrorKey(e);
      setError(t(key === "offline" ? "newTrip.errors.offline" : "newTrip.errors.generic"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        <ScrollView
          keyboardShouldPersistTaps="handled"
          contentContainerStyle={{ padding: 20, gap: 20 }}
        >
          <View
            style={{ flexDirection: "row", justifyContent: "space-between", alignItems: "center" }}
          >
            <IconButton
              accessibilityLabel={t("newTrip.close")}
              variant="secondary"
              onPress={() => router.back()}
            >
              <Text variant="bodyStrong">{"×"}</Text>
            </IconButton>
          </View>

          <View style={{ gap: 8 }}>
            <Text variant="display" heading>
              {t("newTrip.title")}
            </Text>
            <Text color="muted">{t("newTrip.destinationHint")}</Text>
            <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
              {suggestedDestinations.map((d) => {
                const picked = stops.some((s) => s.name === d.name);
                return (
                  <Chip
                    key={d.name}
                    label={d.name}
                    selected={picked}
                    accessibilityHint={picked ? t("newTrip.remove", { place: d.name }) : undefined}
                    onPress={() =>
                      setStops((s) => (picked ? s.filter((x) => x.name !== d.name) : [...s, d]))
                    }
                  />
                );
              })}
            </View>
          </View>

          <View style={{ gap: 12 }}>
            <View
              style={{
                flexDirection: "row",
                justifyContent: "space-between",
                alignItems: "center",
              }}
            >
              <Text variant="title" heading>
                {t("newTrip.when")}
              </Text>
              <View style={{ flexDirection: "row", gap: 8 }}>
                <Chip
                  label={t("newTrip.dates")}
                  selected={mode === "dates"}
                  onPress={() => setMode("dates")}
                />
                <Chip
                  label={t("newTrip.flexible")}
                  selected={mode === "flexible"}
                  onPress={() => setMode("flexible")}
                />
              </View>
            </View>
            {mode === "dates" ? (
              <DateRangePicker value={range} onChange={setRange} locale={tag} />
            ) : (
              <View
                style={{ backgroundColor: colors.white, borderRadius: 22, padding: 14, gap: 8 }}
              >
                <Text variant="bodyStrong">{t("newTrip.flexibleDays")}</Text>
                <View
                  style={{
                    flexDirection: "row",
                    alignItems: "center",
                    justifyContent: "space-between",
                  }}
                >
                  <IconButton
                    accessibilityLabel={t("newTrip.fewerDays")}
                    variant="secondary"
                    disabled={days <= 1}
                    onPress={() => setDays((d) => Math.max(1, d - 1))}
                  >
                    <Text variant="title">{"−"}</Text>
                  </IconButton>
                  <Text variant="title" accessibilityLiveRegion="polite">
                    {t("newTrip.daysCount", { count: days })}
                  </Text>
                  <IconButton
                    accessibilityLabel={t("newTrip.moreDays")}
                    variant="secondary"
                    onPress={() => setDays((d) => Math.min(60, d + 1))}
                  >
                    <Text variant="title">{"+"}</Text>
                  </IconButton>
                </View>
              </View>
            )}
          </View>

          <Input
            label={t("newTrip.nameLabel")}
            hint={t("newTrip.nameHint")}
            value={name}
            onChangeText={setName}
            maxLength={80}
          />
          <Input
            label={t("newTrip.currencyLabel")}
            hint={t("newTrip.currencyHint")}
            value={currency}
            onChangeText={(v) => setCurrency(v.toUpperCase())}
            autoCapitalize="characters"
            autoCorrect={false}
            maxLength={3}
          />

          {error ? (
            <Text color="vermilion" accessibilityLiveRegion="polite">
              {error}
            </Text>
          ) : null}
          <Button label={t("newTrip.save")} fullWidth loading={busy} onPress={save} />
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
