import { useLocalSearchParams, useRouter } from "expo-router";
import { track } from "@/analytics/track";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { signInErrorKey, useAuth, withDetail } from "@/auth";
import { reportError } from "@/logging/remoteLog";
import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { useTheme } from "@/theme";
import { countryName, suggestedDestinations, type Destination } from "@/trips/destinations";
import { DateRangePicker } from "@/trips/DateRangePicker";
import type { DateRange } from "@/trips/dates";
import { Button, Chip, IconButton, Input, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Mode = "dates" | "flexible";

export default function NewTripScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  const { api } = useAuth();
  const { locale: tag, currency: homeCurrency } = useProfile();

  const { destination } = useLocalSearchParams<{ destination?: string }>();
  const [stops, setStops] = useState<Destination[]>(() =>
    suggestedDestinations.filter((d) => d.name === destination),
  );
  const [mode, setMode] = useState<Mode>("dates");
  const [range, setRange] = useState<DateRange>({});
  const [days, setDays] = useState(7);
  const [name, setName] = useState("");
  // Until the person types one, the trip's money follows their home currency.
  const [typedCurrency, setCurrency] = useState<string | undefined>();
  const currency = typedCurrency ?? homeCurrency;
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const first = stops[0];
  const { region } = useProfile();
  const abroad = stops.find((d) => region && d.countryCode !== region);
  const foreign = abroad ? countryName(abroad.countryCode) : undefined;

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
      track("trip_created");
      router.replace("/");
    } catch (e) {
      reportError("create-trip", e);
      const key = signInErrorKey(e);
      setError(
        withDetail(
          t(key === "offline" ? "newTrip.errors.offline" : "newTrip.errors.generic"),
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
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        <ScrollView
          keyboardShouldPersistTaps="handled"
          contentContainerStyle={{ padding: 20, gap: 20 }}
        >
          <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
            <IconButton
              accessibilityLabel={t("newTrip.close")}
              variant="secondary"
              onPress={() => router.back()}
            >
              <Icon name="close" color={colors.ink} />
            </IconButton>
            <Text variant="caption" color="muted" style={{ flex: 1, textAlign: "center" }}>
              {t("newTrip.heading")}
            </Text>
            <View style={{ width: 44 }} />
          </View>

          <View style={{ gap: 12 }}>
            <Text variant="display" heading style={{ fontSize: 32 }}>
              {t("newTrip.title")}
            </Text>
            <View
              style={{
                flexDirection: "row",
                flexWrap: "wrap",
                alignItems: "center",
                gap: 8,
                minHeight: 56,
                padding: 8,
                borderRadius: 20,
                borderWidth: 2,
                borderColor: colors.ink,
                backgroundColor: colors.white,
              }}
            >
              {stops.map((d) => (
                <Pressable
                  key={d.name}
                  accessibilityRole="button"
                  accessibilityLabel={t("newTrip.remove", { place: d.name })}
                  hitSlop={6}
                  onPress={() => setStops((s) => s.filter((x) => x.name !== d.name))}
                  style={{
                    flexDirection: "row",
                    alignItems: "center",
                    gap: 8,
                    backgroundColor: colors.indigo,
                    borderRadius: 14,
                    paddingVertical: 8,
                    paddingLeft: 14,
                    paddingRight: 8,
                  }}
                >
                  <Text variant="bodyStrong" color="white">
                    {d.name}
                  </Text>
                  <Icon name="close" size={14} color={colors.white} />
                </Pressable>
              ))}
              {stops.length === 0 ? <Text color="muted">{t("newTrip.addStop")}</Text> : null}
            </View>
            {foreign ? (
              <View
                style={{
                  backgroundColor: `${colors.indigo}1A`,
                  borderRadius: 16,
                  padding: 14,
                }}
              >
                <Text variant="body" style={{ color: colors.indigo }}>
                  {t("newTrip.international", { country: foreign })}
                </Text>
              </View>
            ) : null}
            <Text color="muted">{t("newTrip.destinationHint")}</Text>
            <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
              {suggestedDestinations
                .filter((d) => !stops.some((s) => s.name === d.name))
                .map((d) => (
                  <Chip
                    key={d.name}
                    label={d.name}
                    selected={false}
                    onPress={() => setStops((s) => [...s, d])}
                  />
                ))}
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
              <View
                style={{
                  flexDirection: "row",
                  backgroundColor: colors.white,
                  borderRadius: 22,
                  padding: 4,
                }}
              >
                {(["dates", "flexible"] as const).map((m) => (
                  <Pressable
                    key={m}
                    accessibilityRole="button"
                    accessibilityState={{ selected: mode === m }}
                    hitSlop={4}
                    onPress={() => setMode(m)}
                    style={{
                      minHeight: 40,
                      paddingHorizontal: 16,
                      borderRadius: 18,
                      justifyContent: "center",
                      backgroundColor: mode === m ? colors.ink : "transparent",
                    }}
                  >
                    <Text variant="bodyStrong" color={mode === m ? "white" : "ink"}>
                      {t(`newTrip.${m}`)}
                    </Text>
                  </Pressable>
                ))}
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
                    <Icon name="plus" color={colors.ink} />
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
          <Button
            label={t("newTrip.save")}
            variant="dark"
            fullWidth
            loading={busy}
            onPress={save}
          />
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
