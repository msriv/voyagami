import { formatDate } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useMemo, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError } from "@voyagami/api-client";

import { localFirst } from "@/local/localFirst";
import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { useTripChanges } from "@/trips/useTripChanges";
import { daysInRange, isoToDate } from "@/trips/dates";
import { BestOrder } from "@/trips/BestOrder";
import { DraggableStop } from "@/trips/DraggableStop";
import { displayTime, isValidTime, planMove, stopsOn, type Move, type Stop } from "@/trips/stops";
import type { Trip } from "@/trips/summary";
import { Button, Card, Chip, IconButton, Input, Sheet, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type State =
  | { kind: "loading" }
  | { kind: "ready"; trip: Trip; stops: Stop[] }
  | { kind: "notFound" }
  | { kind: "error" };

export default function ItineraryScreen() {
  const { t } = useTranslation();
  const { colors, radius, hitArea } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [state, setState] = useState<State>({ kind: "loading" });
  const [day, setDay] = useState<string | undefined>();
  const [editing, setEditing] = useState<Stop | undefined>();
  const [time, setTime] = useState("");
  const [duration, setDuration] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      await localFirst(
        `stops:${id}`,
        async () => {
          const [trip, { stops }] = await Promise.all([
            api.unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } })),
            api.unwrap(api.GET("/v1/trips/{tripId}/stops", { params: { path: { tripId: id } } })),
          ]);
          return { trip, stops };
        },
        (d) => setState({ kind: "ready", ...d }),
      );
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  // Also reloads on coming back from adding a place.
  useTripChanges(id, () => void load());

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  const { start, end } = state.kind === "ready" ? state.trip.dates : {};
  const days = useMemo(() => (start && end ? daysInRange(start, end) : []), [start, end]);
  const current = day && days.includes(day) ? day : days[0];

  const back = (
    <IconButton
      accessibilityLabel={t("itinerary.back")}
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
  // The stop being edited, as it is now: a move changes its day and place in the list.
  const live = editing ? (stops.find((s) => s.id === editing.id) ?? editing) : undefined;

  function open(next: Stop) {
    setError(undefined);
    setEditing(next);
    setTime(next?.startLocal ?? "");
    setDuration(next?.durationMin ? String(next.durationMin) : "");
    setNote(next?.note ?? "");
  }

  function fail(e: unknown) {
    setError(
      t(e instanceof NetworkError ? "itinerary.errors.offline" : "itinerary.errors.generic"),
    );
  }

  async function save() {
    const minutes = duration.trim() === "" ? 0 : Number(duration);
    if (!editing) return;
    if (!isValidTime(time.trim())) return setError(t("itinerary.errors.time"));
    if (!Number.isInteger(minutes) || minutes < 0 || minutes > 1440) {
      return setError(t("itinerary.errors.duration"));
    }
    setBusy(true);
    setError(undefined);
    try {
      const saved = await api.unwrap(
        api.PATCH("/v1/trips/{tripId}/stops/{stopId}", {
          params: { path: { tripId: id, stopId: editing.id } },
          body: { startLocal: time.trim(), durationMin: minutes, note: note.trim() },
        }),
      );
      setState({ kind: "ready", trip, stops: [...stops.filter((s) => s.id !== saved.id), saved] });
      setEditing(undefined);
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  /** Moves a stop and shows it at once; puts the list back if the server says no. */
  async function move(stop: Stop, how: Move) {
    const plan = planMove(stops, stop, how);
    if (!plan) return;
    setError(undefined);
    setState({ kind: "ready", trip, stops: plan.local });
    try {
      const saved = await api.unwrap(
        api.PATCH("/v1/trips/{tripId}/stops/{stopId}", {
          params: { path: { tripId: id, stopId: stop.id } },
          body: plan.body,
        }),
      );
      setState((s) =>
        s.kind === "ready"
          ? { ...s, stops: s.stops.map((x) => (x.id === saved.id ? saved : x)) }
          : s,
      );
    } catch (e) {
      setState({ kind: "ready", trip, stops });
      fail(e);
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
                <DraggableStop
                  key={s.id}
                  enabled={canEdit}
                  label={t("itinerary.dragHandle", { name: placeName })}
                  onDrop={(rows) =>
                    void move(s, {
                      kind: "to",
                      index: todays.findIndex((x) => x.id === s.id) + rows,
                    })
                  }
                >
                  <Card>
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
                      onPress={() => open(s)}
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
                    <Button
                      label={t("itinerary.comments", { name: placeName })}
                      variant="secondary"
                      onPress={() =>
                        router.push({
                          pathname: "/trips/[id]/comments",
                          params: { id, stopId: s.id, title: placeName },
                        })
                      }
                    />
                  </Card>
                </DraggableStop>
              );
            })}
            {canEdit && current ? (
              <BestOrder
                tripId={id}
                date={current}
                stops={stops}
                onStops={(next) => setState({ kind: "ready", trip, stops: next })}
              />
            ) : null}
            {canEdit ? (
              <Button
                label={t("itinerary.add")}
                onPress={() => router.push(`/trips/${id}/add-place?date=${current}`)}
              />
            ) : null}
          </>
        )}
      </ScrollView>

      <Sheet
        visible={editing !== undefined}
        onClose={() => setEditing(undefined)}
        title={t("itinerary.editTitle")}
        closeLabel={t("itinerary.close")}
        closeIcon={<Icon name="close" color={colors.ink} />}
      >
        <View style={{ gap: 12, paddingBottom: 12 }}>
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
          <Button label={t("itinerary.save")} loading={busy} onPress={() => void save()} />
          {live ? (
            <View style={{ gap: 8 }}>
              <Text variant="bodyStrong">{t("itinerary.moveHeading")}</Text>
              <Button
                label={t("itinerary.moveUp")}
                variant="secondary"
                disabled={!planMove(stops, live, { kind: "up" })}
                onPress={() => void move(live, { kind: "up" })}
              />
              <Button
                label={t("itinerary.moveDown")}
                variant="secondary"
                disabled={!planMove(stops, live, { kind: "down" })}
                onPress={() => void move(live, { kind: "down" })}
              />
              <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                accessibilityLabel={t("itinerary.moveToDay")}
                contentContainerStyle={{ gap: 8 }}
              >
                {days.map((d, i) =>
                  d === live.date ? null : (
                    <Chip
                      key={d}
                      label={t("itinerary.moveToDayN", { n: i + 1 })}
                      selected={false}
                      onPress={() => void move(live, { kind: "day", date: d })}
                    />
                  ),
                )}
              </ScrollView>
            </View>
          ) : null}
          {live ? (
            <Button
              label={t("itinerary.remove")}
              variant="secondary"
              loading={busy}
              onPress={() => void remove(live)}
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
