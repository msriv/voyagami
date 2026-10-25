import { formatDate } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { daysInRange, isoToDate } from "@/trips/dates";
import { displayTime, isValidTime, stopsOn, type Stop } from "@/trips/stops";
import type { Trip } from "@/trips/summary";
import { Button, Card, Chip, IconButton, Input, Sheet, Text } from "@/ui";

type State =
  | { kind: "loading" }
  | { kind: "ready"; trip: Trip; stops: Stop[] }
  | { kind: "notFound" }
  | { kind: "error" };

/** What the stop sheet is doing: adding to a day, or editing one stop. */
type Editing = { kind: "add" } | { kind: "edit"; stop: Stop } | undefined;

export default function ItineraryScreen() {
  const { t } = useTranslation();
  const { colors, radius, hitArea } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [state, setState] = useState<State>({ kind: "loading" });
  const [day, setDay] = useState<string | undefined>();
  const [editing, setEditing] = useState<Editing>();
  const [name, setName] = useState("");
  const [time, setTime] = useState("");
  const [duration, setDuration] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const [trip, { stops }] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/trips/{tripId}/stops", { params: { path: { tripId: id } } })),
      ]);
      setState({ kind: "ready", trip, stops });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  useEffect(() => {
    void load();
  }, [load]);

  const { start, end } = state.kind === "ready" ? state.trip.dates : {};
  const days = useMemo(() => (start && end ? daysInRange(start, end) : []), [start, end]);
  const current = day && days.includes(day) ? day : days[0];

  const back = (
    <IconButton
      accessibilityLabel={t("itinerary.back")}
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
            <Text>
              {state.kind === "notFound" ? t("itinerary.notFound") : t("itinerary.loadError")}
            </Text>
            {state.kind === "error" ? (
              <Button
                label={t("itinerary.retry")}
                variant="secondary"
                onPress={() => void load()}
              />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { trip, stops } = state;
  const canEdit = trip.role !== "viewer";
  const todays = current ? stopsOn(stops, current) : [];

  function open(next: Editing) {
    setError(undefined);
    setEditing(next);
    const s = next?.kind === "edit" ? next.stop : undefined;
    setName(s?.place.name ?? "");
    setTime(s?.startLocal ?? "");
    setDuration(s?.durationMin ? String(s.durationMin) : "");
    setNote(s?.note ?? "");
  }

  function fail(e: unknown) {
    setError(
      t(e instanceof NetworkError ? "itinerary.errors.offline" : "itinerary.errors.generic"),
    );
  }

  async function save() {
    const minutes = duration.trim() === "" ? 0 : Number(duration);
    if (editing?.kind === "add" && !name.trim()) return setError(t("itinerary.errors.name"));
    if (!isValidTime(time.trim())) return setError(t("itinerary.errors.time"));
    if (!Number.isInteger(minutes) || minutes < 0 || minutes > 1440) {
      return setError(t("itinerary.errors.duration"));
    }
    setBusy(true);
    setError(undefined);
    try {
      let saved: Stop;
      if (editing?.kind === "edit") {
        saved = await api.unwrap(
          api.PATCH("/v1/trips/{tripId}/stops/{stopId}", {
            params: { path: { tripId: id, stopId: editing.stop.id } },
            body: { startLocal: time.trim(), durationMin: minutes, note: note.trim() },
          }),
        );
      } else if (current) {
        // shortcut: a stop added by name alone has no map position (0, 0), until place search and the manual tab (VOY-124) supply real coordinates.
        saved = await api.unwrap(
          api.POST("/v1/trips/{tripId}/stops", {
            params: { path: { tripId: id } },
            body: {
              date: current,
              place: { name: name.trim(), lat: 0, lng: 0 },
              ...(time.trim() ? { startLocal: time.trim() } : null),
              ...(minutes ? { durationMin: minutes } : null),
              ...(note.trim() ? { note: note.trim() } : null),
            },
          }),
        );
      } else return;
      setState({ kind: "ready", trip, stops: [...stops.filter((s) => s.id !== saved.id), saved] });
      setEditing(undefined);
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function remove(stop: Stop) {
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/stops/{stopId}", {
          params: { path: { tripId: id, stopId: stop.id } },
        }),
      );
      setState({ kind: "ready", trip, stops: stops.filter((s) => s.id !== stop.id) });
      setEditing(undefined);
    } catch (e) {
      fail(e);
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
            {t("itinerary.title")}
          </Text>
        </View>
        <Text color="muted">{trip.title}</Text>

        {days.length === 0 ? (
          <Text>{t("itinerary.noDates")}</Text>
        ) : (
          <>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              accessibilityLabel={t("itinerary.daysLabel")}
              contentContainerStyle={{ gap: 8 }}
            >
              {days.map((d, i) => (
                <Chip
                  key={d}
                  label={t("itinerary.dayTab", {
                    n: i + 1,
                    date: formatDate(isoToDate(d), locale, { timeZone: "UTC" }),
                  })}
                  selected={d === current}
                  onPress={() => setDay(d)}
                />
              ))}
            </ScrollView>

            {todays.length === 0 ? (
              <Text color="muted">
                {t(canEdit ? "itinerary.emptyDay" : "itinerary.emptyDayViewer")}
              </Text>
            ) : null}
            {todays.map((s) => {
              const placeName = s.place.name ?? "";
              const when = s.startLocal ? displayTime(s.startLocal, locale) : "";
              return (
                <Card key={s.id}>
                  <Pressable
                    accessibilityRole="button"
                    accessibilityLabel={
                      canEdit
                        ? t("itinerary.editStop", { name: placeName })
                        : when
                          ? t("itinerary.stopLabel", { name: placeName, time: when })
                          : t("itinerary.stopLabelNoTime", { name: placeName })
                    }
                    disabled={!canEdit}
                    onPress={() => open({ kind: "edit", stop: s })}
                    style={{ minHeight: hitArea.min, gap: 2, borderRadius: radius.button }}
                  >
                    {when ? <Text variant="caption">{when}</Text> : null}
                    <Text variant="bodyStrong">{placeName}</Text>
                    {s.durationMin ? (
                      <Text variant="caption">
                        {t("itinerary.minutes", { count: s.durationMin })}
                      </Text>
                    ) : null}
                    {s.note ? <Text color="muted">{s.note}</Text> : null}
                  </Pressable>
                </Card>
              );
            })}
            {canEdit ? (
              <Button label={t("itinerary.add")} onPress={() => open({ kind: "add" })} />
            ) : null}
          </>
        )}
      </ScrollView>

      <Sheet
        visible={editing !== undefined}
        onClose={() => setEditing(undefined)}
        title={t(editing?.kind === "edit" ? "itinerary.editTitle" : "itinerary.addTitle")}
        closeLabel={t("itinerary.close")}
        closeIcon={<Text>{"×"}</Text>}
      >
        <View style={{ gap: 12, paddingBottom: 12 }}>
          {editing?.kind === "add" ? (
            <Input
              label={t("itinerary.nameLabel")}
              value={name}
              onChangeText={setName}
              maxLength={200}
            />
          ) : null}
          <Input
            label={t("itinerary.timeLabel")}
            value={time}
            onChangeText={setTime}
            maxLength={5}
            keyboardType="numbers-and-punctuation"
          />
          <Input
            label={t("itinerary.durationLabel")}
            value={duration}
            onChangeText={setDuration}
            maxLength={4}
            keyboardType="number-pad"
          />
          <Input
            label={t("itinerary.noteLabel")}
            value={note}
            onChangeText={setNote}
            maxLength={2000}
            multiline
          />
          <Button
            label={t(editing?.kind === "edit" ? "itinerary.save" : "itinerary.addConfirm")}
            loading={busy}
            onPress={() => void save()}
          />
          {editing?.kind === "edit" ? (
            <Button
              label={t("itinerary.remove")}
              variant="secondary"
              loading={busy}
              onPress={() => void remove(editing.stop)}
            />
          ) : null}
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
