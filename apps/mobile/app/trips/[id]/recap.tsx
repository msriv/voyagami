import { formatDate, formatDistance, formatMoney } from "@voyagami/locale";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { useTheme } from "@/theme";
import { isoToDate } from "@/trips/dates";
import { Button, Card, IconButton, Text } from "@/ui";

type Recap = components["schemas"]["Recap"];
type Trip = components["schemas"]["Trip"];

type State =
  | { kind: "loading" }
  | { kind: "ready"; recap: Recap; trip: Trip; names: Record<string, string> }
  | { kind: "notFound" }
  | { kind: "error" };

/** The end-of-trip summary: stamp, numbers, money and a few journal highlights. */
export default function RecapScreen() {
  const { t } = useTranslation();
  const { colors, radius } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { locale, units } = useProfile();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [state, setState] = useState<State>({ kind: "loading" });

  const load = useCallback(async () => {
    try {
      const path = { params: { path: { tripId: id } } };
      const [recap, trip, { members }] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}/recap", path)),
        api.unwrap(api.GET("/v1/trips/{tripId}", path)),
        api.unwrap(api.GET("/v1/trips/{tripId}/members", path)),
      ]);
      const names = Object.fromEntries(members.map((m) => [m.uid, m.displayName ?? ""]));
      setState({ kind: "ready", recap, trip, names });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  const back = (
    <IconButton
      accessibilityLabel={t("recap.back")}
      variant="secondary"
      onPress={() => router.back()}
    >
      <Text variant="bodyStrong">{"‹"}</Text>
    </IconButton>
  );

  if (state.kind !== "ready") {
    return (
      <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi, padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row" }}>{back}</View>
        {state.kind === "loading" ? null : (
          <View style={{ gap: 12 }}>
            <Text>{state.kind === "notFound" ? t("recap.notFound") : t("recap.loadError")}</Text>
            {state.kind === "error" ? (
              <Button label={t("recap.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { recap, trip, names } = state;
  const day = (iso: string) => formatDate(isoToDate(iso), locale, { timeZone: "UTC" });
  const { start, end } = trip.dates;
  const range =
    start && end
      ? t("trip.dateRange", {
          start: day(start),
          end: formatDate(isoToDate(end), locale, { timeZone: "UTC", withYear: true }),
        })
      : "";
  const empty = recap.stops === 0 && recap.highlights.length === 0 && recap.spentMinor === 0;
  const money = (n: number) => formatMoney(n, recap.currency, locale);
  const stats = [
    t("recap.days", { count: recap.days }),
    t("recap.stops", { count: recap.stops }),
    t("recap.friends", { count: recap.crew }),
  ];

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          {back}
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("recap.title")}
          </Text>
        </View>

        <View
          accessible
          style={{
            gap: 4,
            padding: 20,
            borderRadius: radius.card,
            borderWidth: 2,
            borderStyle: "dashed",
            borderColor: colors.indigo,
            backgroundColor: colors.white,
          }}
        >
          <Text variant="caption">{t("recap.stamped")}</Text>
          <Text variant="display" heading>
            {trip.title}
          </Text>
          {range ? <Text color="muted">{range}</Text> : null}
        </View>

        {empty ? (
          <Card>
            <Text>{t("recap.empty")}</Text>
          </Card>
        ) : (
          <>
            <View
              accessible
              accessibilityLabel={`${t("recap.stats")}: ${stats.join(", ")}`}
              style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}
            >
              {stats.map((label) => (
                <View
                  key={label}
                  style={{
                    flexGrow: 1,
                    padding: 14,
                    borderRadius: radius.card,
                    backgroundColor: colors.matcha,
                  }}
                >
                  <Text variant="bodyStrong">{label}</Text>
                </View>
              ))}
            </View>
            {recap.distanceMeters > 0 ? (
              <Text color="muted">
                {t("recap.distance", {
                  distance: formatDistance(recap.distanceMeters, units.distance, locale),
                })}
              </Text>
            ) : null}

            <Card>
              <View style={{ gap: 10 }}>
                <Text variant="bodyStrong">
                  {t("recap.spent", { amount: money(recap.spentMinor) })}
                </Text>
                {recap.remainingMinor != null ? (
                  <Text color="muted">
                    {t(recap.remainingMinor >= 0 ? "recap.underBudget" : "recap.overBudget", {
                      amount: money(Math.abs(recap.remainingMinor)),
                    })}
                  </Text>
                ) : null}
                {recap.settledUp != null ? (
                  <Text color="muted">
                    {t(recap.settledUp ? "recap.settled" : "recap.notSettled")}
                  </Text>
                ) : null}
                {recap.favouriteDay ? (
                  <View style={{ gap: 2 }}>
                    <Text variant="caption">{t("recap.favouriteDay")}</Text>
                    <Text variant="bodyStrong">{day(recap.favouriteDay.date)}</Text>
                  </View>
                ) : null}
              </View>
            </Card>

            {recap.highlights.length > 0 ? (
              <View style={{ gap: 8 }}>
                <Text variant="title" heading>
                  {t("recap.highlights")}
                </Text>
                {recap.highlights.map((h) => (
                  <Card key={`${h.date}-${h.authorUid}`}>
                    <Text variant="caption">
                      {day(h.date)}
                      {names[h.authorUid] ? ` · ${names[h.authorUid]}` : ""}
                    </Text>
                    <Text>{h.text}</Text>
                  </Card>
                ))}
              </View>
            ) : null}
          </>
        )}

        <Button
          label={t("recap.play")}
          variant="secondary"
          onPress={() => router.push(`/trips/${id}/recap-player`)}
        />
        <Button label={t("recap.planNext")} onPress={() => router.push("/trips/new")} />
      </ScrollView>
    </SafeAreaView>
  );
}
