import { formatDate, formatTime } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import type { Trip } from "@/trips/summary";
import { Button, Card, IconButton, Sheet, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Booking = components["schemas"]["Booking"];

type State =
  | { kind: "loading" }
  | { kind: "ready"; trip: Trip; bookings: Booking[] }
  | { kind: "notFound" }
  | { kind: "error" };

/** Reservations and tickets on a trip, soonest first, then the ones without a time. */
export default function BookingsScreen() {
  const { t } = useTranslation();
  const { colors, radius, hitArea } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [state, setState] = useState<State>({ kind: "loading" });
  const [open, setOpen] = useState<Booking | undefined>();
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const [trip, { bookings }] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/trips/{tripId}/bookings", { params: { path: { tripId: id } } })),
      ]);
      setState({ kind: "ready", trip, bookings });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  // Also reloads on coming back from adding a booking.
  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  const back = (
    <IconButton
      accessibilityLabel={t("bookings.back")}
      variant="secondary"
      onPress={() => router.back()}
    >
      <Icon name="back" color={colors.ink} />
    </IconButton>
  );

  if (state.kind !== "ready") {
    return (
      <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi, padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row" }}>{back}</View>
        {state.kind === "loading" ? null : (
          <View style={{ gap: 12 }}>
            <Text>
              {state.kind === "notFound" ? t("bookings.notFound") : t("bookings.loadError")}
            </Text>
            {state.kind === "error" ? (
              <Button label={t("bookings.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { trip, bookings } = state;
  const canEdit = trip.role !== "viewer";
  const sorted = [...bookings].sort((a, b) =>
    a.startsAt && b.startsAt
      ? a.startsAt.localeCompare(b.startsAt)
      : a.startsAt
        ? -1
        : b.startsAt
          ? 1
          : 0,
  );

  function when(b: Booking): string {
    if (!b.startsAt) return t("bookings.noTime");
    const at = new Date(b.startsAt);
    const timeZone = b.startTimeZone ?? trip.timeZone;
    return `${formatDate(at, locale, { timeZone })}, ${formatTime(at, locale, { timeZone })}`;
  }

  async function remove(booking: Booking) {
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/bookings/{bookingId}", {
          params: { path: { tripId: id, bookingId: booking.id } },
        }),
      );
      setState({ kind: "ready", trip, bookings: bookings.filter((b) => b.id !== booking.id) });
      setOpen(undefined);
    } catch (e) {
      setError(
        t(e instanceof NetworkError ? "bookings.errors.offline" : "bookings.errors.generic"),
      );
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
            {t("bookings.title")}
          </Text>
        </View>
        <Text color="muted">{trip.title}</Text>

        {sorted.length === 0 ? (
          <Text color="muted">{t(canEdit ? "bookings.empty" : "bookings.emptyViewer")}</Text>
        ) : null}
        {sorted.map((b) => (
          <Card key={b.id}>
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={t("bookings.bookingLabel", {
                kind: t(`bookings.kinds.${b.kind}`),
                name: b.title,
                when: when(b),
              })}
              disabled={!canEdit}
              onPress={() => {
                setError(undefined);
                setOpen(b);
              }}
              style={{ minHeight: hitArea.min, gap: 2, borderRadius: radius.button }}
            >
              <Text variant="caption">{t(`bookings.kinds.${b.kind}`).toUpperCase()}</Text>
              <Text variant="bodyStrong">{b.title}</Text>
              {b.provider ? <Text color="muted">{b.provider}</Text> : null}
              <Text variant="caption">{when(b)}</Text>
            </Pressable>
          </Card>
        ))}

        {canEdit ? (
          <>
            <Button
              label={t("bookings.add")}
              onPress={() => router.push(`/trips/${id}/add-booking`)}
            />
            <Button
              label={t("bookings.import")}
              variant="secondary"
              onPress={() => router.push(`/trips/${id}/import-booking`)}
            />
          </>
        ) : null}
      </ScrollView>

      <Sheet
        visible={open !== undefined}
        onClose={() => setOpen(undefined)}
        title={open?.title ?? t("bookings.title")}
        closeLabel={t("bookings.close")}
        closeIcon={<Icon name="close" color={colors.ink} />}
      >
        <View style={{ gap: 12, paddingBottom: 12 }}>
          <Button
            label={t("bookings.remove")}
            variant="secondary"
            loading={busy}
            onPress={() => open && void remove(open)}
          />
          {error ? (
            <Text color="vermilion" accessibilityLiveRegion="polite">
              {error}
            </Text>
          ) : null}
        </View>
      </Sheet>
    </SafeAreaView>
  );
}
