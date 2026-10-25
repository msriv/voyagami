import { formatDate } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useMemo, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { daysInRange, isoToDate } from "@/trips/dates";
import { stopsOn, type Stop } from "@/trips/stops";
import type { Trip } from "@/trips/summary";
import { Button, Card, Chip, IconButton, Input, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Entry = components["schemas"]["JournalEntry"];

const moods = ["happy", "excited", "calm", "grateful", "tired", "meh"] as const;
type Mood = (typeof moods)[number];

type State =
  | { kind: "loading" }
  | { kind: "ready"; trip: Trip; entries: Entry[]; stops: Stop[]; me: string }
  | { kind: "notFound" }
  | { kind: "error" };

/**
 * One day's page: your own entry to write, the day's plan underneath it, and what the crew shared.
 * shortcut: no photos, voice or autosave of drafts, and crew entries show no names, until the
 * members list and media exist.
 */
function DayEntry({
  tripId,
  date,
  mine,
  plan,
  canEdit,
  onChange,
}: {
  tripId: string;
  date: string;
  mine: Entry | undefined;
  plan: Stop[];
  canEdit: boolean;
  onChange: (entry: Entry | undefined) => void;
}) {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const [text, setText] = useState(mine?.text ?? "");
  const [mood, setMood] = useState<Mood | undefined>(mine?.mood as Mood | undefined);
  const [priv, setPriv] = useState(mine?.visibility === "private");
  const [error, setError] = useState<string | undefined>();
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);

  const fail = (e: unknown) =>
    setError(t(e instanceof NetworkError ? "journal.errors.offline" : "journal.errors.generic"));

  async function save() {
    if (!text.trim()) return setError(t("journal.errors.empty"));
    setBusy(true);
    setError(undefined);
    setSaved(false);
    try {
      const entry = await api.unwrap(
        api.PUT("/v1/trips/{tripId}/journal/{date}", {
          params: { path: { tripId, date } },
          body: {
            text: text.trim(),
            ...(mood ? { mood } : null),
            visibility: priv ? "private" : "crew",
            stopIds: plan.slice(0, 20).map((s) => s.id),
          },
        }),
      );
      onChange(entry);
      setSaved(true);
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/journal/{date}", { params: { path: { tripId, date } } }),
      );
      onChange(undefined);
      setText("");
      setMood(undefined);
      setPriv(false);
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  if (!canEdit) return null;
  return (
    <View style={{ gap: 12 }}>
      <View style={{ gap: 8 }}>
        <Text variant="bodyStrong">{t("journal.moodLabel")}</Text>
        <View style={{ flexDirection: "row", gap: 8 }}>
          {moods.map((m) => (
            <Pressable
              key={m}
              accessibilityRole="button"
              accessibilityLabel={t(`journal.moods.${m}`)}
              accessibilityState={{ selected: m === mood }}
              onPress={() => setMood(m === mood ? undefined : m)}
              style={{
                flex: 1,
                minHeight: 48,
                borderRadius: 16,
                alignItems: "center",
                justifyContent: "center",
                backgroundColor: m === mood ? colors.vermilion : colors.white,
              }}
            >
              <Text variant={m === mood ? "bodyStrong" : "body"}>{t(`journal.moods.${m}`)}</Text>
            </Pressable>
          ))}
        </View>
      </View>
      <View style={{ backgroundColor: colors.white, borderRadius: 22, padding: 16, gap: 10 }}>
        <Input
          label={t("journal.textLabel")}
          value={text}
          onChangeText={(v) => {
            setText(v);
            setSaved(false);
          }}
          maxLength={10000}
          multiline
        />
        <View style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
          <Chip label={t("journal.justMe")} selected={priv} onPress={() => setPriv(!priv)} />
          <Text variant="caption" style={{ flex: 1 }}>
            {t(priv ? "journal.privateHint" : "journal.crewHint")}
          </Text>
        </View>
      </View>
      <Button label={t("journal.save")} loading={busy} onPress={() => void save()} />
      {mine ? (
        <Button
          label={t("journal.remove")}
          variant="secondary"
          loading={busy}
          onPress={() => void remove()}
        />
      ) : null}
      {saved ? <Text accessibilityLiveRegion="polite">{t("journal.saved")}</Text> : null}
      {error ? (
        <Text color="vermilion" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

export default function JournalScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [state, setState] = useState<State>({ kind: "loading" });
  const [day, setDay] = useState<string | undefined>();

  const load = useCallback(async () => {
    try {
      const [trip, { entries }, { stops }, me] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/trips/{tripId}/journal", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/trips/{tripId}/stops", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/me")),
      ]);
      setState({ kind: "ready", trip, entries, stops, me: me.id });
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

  const { start, end } = state.kind === "ready" ? state.trip.dates : {};
  const days = useMemo(() => (start && end ? daysInRange(start, end) : []), [start, end]);
  const current = day && days.includes(day) ? day : days[0];

  const back = (
    <IconButton
      accessibilityLabel={t("journal.back")}
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
              {state.kind === "notFound" ? t("journal.notFound") : t("journal.loadError")}
            </Text>
            {state.kind === "error" ? (
              <Button label={t("journal.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { trip, entries, stops, me } = state;
  const mine = entries.find((e) => e.date === current && e.authorUid === me);
  const others = entries.filter((e) => e.date === current && e.authorUid !== me);
  const plan = current ? stopsOn(stops, current) : [];

  function setMine(entry: Entry | undefined) {
    setState({
      kind: "ready",
      trip,
      stops,
      me,
      entries: [
        ...entries.filter((e) => !(e.date === current && e.authorUid === me)),
        ...(entry ? [entry] : []),
      ],
    });
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView
        contentContainerStyle={{ padding: 20, gap: 16 }}
        keyboardShouldPersistTaps="handled"
      >
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          {back}
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("journal.title")}
          </Text>
        </View>
        <Text color="muted">{trip.title}</Text>

        {days.length === 0 || !current ? (
          <Text>{t("journal.noDates")}</Text>
        ) : (
          <>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              accessibilityLabel={t("journal.daysLabel")}
              contentContainerStyle={{ gap: 8 }}
            >
              {days.map((d, i) => (
                <Chip
                  key={d}
                  label={t("journal.dayTab", {
                    n: i + 1,
                    date: formatDate(isoToDate(d), locale, { timeZone: "UTC" }),
                  })}
                  selected={d === current}
                  onPress={() => setDay(d)}
                />
              ))}
            </ScrollView>

            {plan.length > 0 ? (
              <Card>
                <Text variant="bodyStrong">{t("journal.plan")}</Text>
                {plan.map((s) => (
                  <Text key={s.id}>{s.place.name ?? ""}</Text>
                ))}
              </Card>
            ) : null}

            <DayEntry
              key={current}
              tripId={id}
              date={current}
              mine={mine}
              plan={plan}
              canEdit={trip.role !== "viewer"}
              onChange={setMine}
            />

            {others.map((e) => (
              <Card key={e.authorUid}>
                <Text variant="caption">{t("journal.fromCrew")}</Text>
                <Text variant="handwriting">{e.text}</Text>
              </Card>
            ))}
          </>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}
