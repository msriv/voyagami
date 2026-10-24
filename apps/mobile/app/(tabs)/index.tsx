import { useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { RefreshControl, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { useProfile } from "@/profile";
import { partitionTrips, todayIso, type Trip } from "@/trips/summary";
import { TripCard } from "@/trips/TripCard";
import { Button, Card, Seal, Text } from "@/ui";

export default function HomeScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { locale, displayName: name } = useProfile();
  const [trips, setTrips] = useState<Trip[] | undefined>();
  const [failed, setFailed] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      const list = await api.unwrap(api.GET("/v1/trips"));
      setTrips(list.trips);
      setFailed(false);
    } catch {
      setFailed(true);
    }
  }, [api]);

  useEffect(() => {
    void load();
  }, [load]);

  const today = todayIso();
  const { upcoming, past } = partitionTrips(trips ?? [], today);
  const empty = trips !== undefined && trips.length === 0;
  const [next, ...later] = upcoming;

  return (
    <SafeAreaView
      style={{ flex: 1, backgroundColor: colors.washi }}
      accessibilityLabel={t("tabs.home")}
    >
      <ScrollView
        contentContainerStyle={{ padding: 20, gap: 18 }}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={() => {
              setRefreshing(true);
              void load().finally(() => setRefreshing(false));
            }}
          />
        }
      >
        <View style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
          <Text variant="accent" style={{ fontSize: 30 }}>
            {t("home.brand")}
          </Text>
          <Seal size={24} />
        </View>

        <View style={{ gap: 4 }}>
          <Text variant="accent" color="muted" style={{ fontSize: 22 }}>
            {name ? t("home.welcomeNamed", { name }) : t("home.welcome")}
          </Text>
          <Text variant="display" heading style={{ fontSize: 38, lineHeight: 40 }}>
            {empty ? t("home.emptyTitle") : t("home.whereNext")}
          </Text>
        </View>

        {trips === undefined && !failed ? <Text color="muted">{t("home.loading")}</Text> : null}
        {failed && trips === undefined ? (
          <Card>
            <Text accessibilityLiveRegion="polite">{t("home.loadError")}</Text>
            <Button label={t("home.retry")} variant="secondary" onPress={() => void load()} />
          </Card>
        ) : null}

        {empty ? (
          <View
            style={{ backgroundColor: colors.vermilion, borderRadius: 24, padding: 20, gap: 4 }}
          >
            <Text variant="title" heading>
              {t("home.emptyPlanTitle")}
            </Text>
            <Text variant="caption" style={{ color: colors.ink, fontWeight: "700" }}>
              {t("home.emptyPlanBody")}
            </Text>
            <Button
              label={t("home.newTrip")}
              variant="primary"
              style={{ marginTop: 12 }}
              onPress={() => router.push("/trips/new")}
            />
          </View>
        ) : null}

        {next ? <TripCard trip={next} locale={locale} today={today} featured /> : null}
        {later.length > 0 ? (
          <View style={{ gap: 10 }}>
            <Text variant="title" heading>
              {t("home.upcoming")}
            </Text>
            {later.map((trip) => (
              <TripCard key={trip.id} trip={trip} locale={locale} today={today} />
            ))}
          </View>
        ) : null}
        {past.length > 0 ? (
          <View style={{ gap: 10 }}>
            <Text variant="title" heading>
              {t("home.past")}
            </Text>
            {past.map((trip) => (
              <TripCard key={trip.id} trip={trip} locale={locale} today={today} />
            ))}
          </View>
        ) : null}

        {trips && trips.length > 0 ? (
          <Button
            label={t("home.newTrip")}
            variant="secondary"
            onPress={() => router.push("/trips/new")}
          />
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}
