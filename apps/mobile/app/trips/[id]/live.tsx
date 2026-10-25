import { formatTime } from "@voyagami/locale";
import * as Location from "expo-location";
import { getLocales } from "expo-localization";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import type { Trip } from "@/trips/summary";
import { Button, Card, IconButton, Text } from "@/ui";

type Shared = components["schemas"]["SharedLocation"];

type State =
  | { kind: "loading" }
  | {
      kind: "ready";
      trip: Trip;
      locations: Shared[];
      names: Record<string, string>;
      me: string;
    }
  | { kind: "notFound" }
  | { kind: "error" };

/** Who is sharing where, and the control to share your own. Sharing is opt-in and ends by itself. */
export default function LiveScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [state, setState] = useState<State>({ kind: "loading" });
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const path = { params: { path: { tripId: id } } };
      const [trip, { locations }, { members }, me] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}", path)),
        api.unwrap(api.GET("/v1/trips/{tripId}/locations", path)),
        api.unwrap(api.GET("/v1/trips/{tripId}/members", path)),
        api.unwrap(api.GET("/v1/me")),
      ]);
      const names = Object.fromEntries(members.map((m) => [m.uid, m.displayName ?? ""]));
      setState({ kind: "ready", trip, locations, names, me: me.id });
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

  const ready = state.kind === "ready" ? state : undefined;
  const mine = ready?.locations.find((l) => l.uid === ready.me);
  const endsAt = mine?.endsAt;

  // While sharing: send a fresh position now and then, and stop when the time is up.
  useEffect(() => {
    if (!endsAt) return;
    let watch: Location.LocationSubscription | undefined;
    let cancelled = false;
    const left = Date.parse(endsAt) - Date.now();
    const timer = setTimeout(() => void load(), Math.max(left, 0) + 1000);
    void Location.watchPositionAsync(
      { accuracy: Location.Accuracy.Balanced, timeInterval: 60_000, distanceInterval: 50 },
      (pos) => {
        void api.PUT("/v1/trips/{tripId}/locations/me", {
          params: { path: { tripId: id } },
          body: {
            lat: pos.coords.latitude,
            lng: pos.coords.longitude,
            accuracyM: Math.round(pos.coords.accuracy ?? 0),
          },
        });
      },
    ).then((w) => (cancelled ? w.remove() : (watch = w)));
    return () => {
      cancelled = true;
      clearTimeout(timer);
      watch?.remove();
    };
  }, [endsAt, api, id, load]);

  const back = (
    <IconButton
      accessibilityLabel={t("live.back")}
      variant="secondary"
      onPress={() => router.back()}
    >
      <Text variant="bodyStrong">{"‹"}</Text>
    </IconButton>
  );

  if (!ready) {
    return (
      <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi, padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row" }}>{back}</View>
        {state.kind === "loading" ? null : (
          <View style={{ gap: 12 }}>
            <Text>{state.kind === "notFound" ? t("live.notFound") : t("live.loadError")}</Text>
            {state.kind === "error" ? (
              <Button label={t("live.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { trip, locations, names, me } = ready;
  const others = locations.filter((l) => l.uid !== me);

  async function start() {
    setBusy(true);
    setError(undefined);
    try {
      const { granted } = await Location.requestForegroundPermissionsAsync();
      if (!granted) return setError(t("live.errors.denied"));
      const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      await api.unwrap(
        api.PUT("/v1/trips/{tripId}/locations/me", {
          params: { path: { tripId: id } },
          body: {
            lat: pos.coords.latitude,
            lng: pos.coords.longitude,
            accuracyM: Math.round(pos.coords.accuracy ?? 0),
            durationMinutes: 60,
          },
        }),
      );
      await load();
    } catch (e) {
      setError(t(e instanceof NetworkError ? "live.errors.offline" : "live.errors.generic"));
    } finally {
      setBusy(false);
    }
  }

  async function stop() {
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/locations/me", { params: { path: { tripId: id } } }),
      );
      await load();
    } catch (e) {
      setError(t(e instanceof NetworkError ? "live.errors.offline" : "live.errors.generic"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          {back}
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("live.title")}
          </Text>
        </View>
        <Text color="muted">{trip.title}</Text>

        <Card>
          <Text variant="bodyStrong">{t(mine ? "live.sharingOn" : "live.shareTitle")}</Text>
          <Text color="muted">
            {mine
              ? t("live.until", { time: formatTime(Date.parse(mine.endsAt), locale) })
              : t("live.explain")}
          </Text>
          <Button
            label={t(mine ? "live.stop" : "live.start")}
            variant={mine ? "secondary" : "primary"}
            loading={busy}
            onPress={() => void (mine ? stop() : start())}
          />
          {error ? (
            <Text color="vermilion" accessibilityLiveRegion="polite">
              {error}
            </Text>
          ) : null}
        </Card>

        <Text variant="bodyStrong">{t("live.crew")}</Text>
        {others.length === 0 ? <Text color="muted">{t("live.nobody")}</Text> : null}
        {others.map((l) => (
          <Card key={l.uid}>
            <Text variant="bodyStrong">{names[l.uid] || t("live.someone")}</Text>
            <Text color="muted">
              {t("live.updated", { time: formatTime(Date.parse(l.updatedAt), locale) })}
            </Text>
          </Card>
        ))}
      </ScrollView>
    </SafeAreaView>
  );
}
