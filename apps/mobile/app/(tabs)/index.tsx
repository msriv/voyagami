import { useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { Pressable, RefreshControl, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { localFirst } from "@/local/localFirst";
import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { StampBoard } from "@/stamps/StampBoard";
import { useTheme } from "@/theme";
import { useProfile } from "@/profile";
import { partitionTrips, todayIso, type Trip } from "@/trips/summary";
import { TripCard } from "@/trips/TripCard";
import { Button, Card, Seal, Text } from "@/ui";
import { Icon } from "@/ui/icons";

export default function HomeScreen() {
  const { t } = useTranslation();
  const { colors, hitArea } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { locale, displayName: name } = useProfile();
  const [trips, setTrips] = useState<Trip[] | undefined>();
  const [failed, setFailed] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      await localFirst(
        "trips",
        async () => (await api.unwrap(api.GET("/v1/trips"))).trips,
        (list) => {
          setTrips(list);
          setFailed(false);
        },
      );
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
  const initial = (name ?? "").trim().charAt(0).toUpperCase();
  const hour = new Date().getHours();
  const greeting = hour < 12 ? "goodMorning" : hour < 18 ? "goodAfternoon" : "goodEvening";

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
        <View
          style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}
        >
          <View style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
            <Text variant="accent" style={{ fontSize: 30 }}>
              {t("home.brand")}
            </Text>
            <Seal size={24} />
          </View>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={t("home.profileButton")}
            onPress={() => router.push("/profile")}
            style={{
              width: hitArea.min,
              height: hitArea.min,
              borderRadius: hitArea.min / 2,
              backgroundColor: colors.ink,
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            {initial ? (
              <Text variant="bodyStrong" color="white">
                {initial}
              </Text>
            ) : (
              <Icon name="person" size={22} color={colors.white} />
            )}
          </Pressable>
        </View>

        <View style={{ gap: 4 }}>
          <Text variant="accent" color="muted" style={{ fontSize: 22 }}>
            {name ? t(`home.${greeting}Named`, { name }) : t(`home.${greeting}`)}
          </Text>
          <Text variant="display" heading style={{ fontSize: 38, lineHeight: 40 }}>
            {empty ? t("home.emptyTitle") : t("home.whereNext")}
          </Text>
        </View>

        <StampBoard />

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
        {next ? (
          <Text variant="caption" color="muted">
            {t("home.quickAdd", { trip: next.title })}
          </Text>
        ) : null}
        {next ? (
          <View style={{ flexDirection: "row", justifyContent: "space-between", gap: 10 }}>
            {(
              [
                ["place", `/trips/${next.id}/add-place`, "plus"],
                ["booking", `/trips/${next.id}/add-booking`, "download"],
                ["todo", `/trips/${next.id}`, "todo"],
                ["expense", `/trips/${next.id}/add-expense`, "people"],
              ] as const
            ).map(([key, href, icon]) => (
              <Pressable
                key={key}
                accessibilityRole="button"
                accessibilityLabel={t(`home.quick.${key}`, { trip: next.title })}
                onPress={() => router.push(href as never)}
                style={{ flex: 1, alignItems: "center", gap: 8, minHeight: hitArea.min }}
              >
                <View
                  style={{
                    width: 58,
                    height: 58,
                    borderRadius: 20,
                    backgroundColor: colors.white,
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                >
                  <Icon name={icon} color={colors.ink} />
                </View>
                <Text variant="caption" style={{ color: colors.ink, fontWeight: "700" }}>
                  {t(`home.quick.${key}Short`)}
                </Text>
              </Pressable>
            ))}
          </View>
        ) : null}
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
        {trips ? (
          <Button
            label={t("home.joinTrip")}
            variant="secondary"
            onPress={() => router.push("/join")}
          />
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}
