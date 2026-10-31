import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import type { components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { daysInRange } from "@/trips/dates";
import { Button, Card, IconButton, Input, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Draft = components["schemas"]["AskDraft"];

/**
 * Ask: describe a trip in plain words and get a day-by-day draft from places we already know.
 * It is a preview. Nothing is added until the person picks places and adds them.
 */
export default function AskScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [prompt, setPrompt] = useState("");
  const [draft, setDraft] = useState<Draft | undefined>();
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState<string | undefined>();

  const key = (day: number, placeId: string) => `${day}:${placeId}`;

  async function ask() {
    if (!prompt.trim()) return setNote(t("ask.empty"));
    setBusy(true);
    setNote(undefined);
    try {
      const d = await api.unwrap(
        api.POST("/v1/trips/{tripId}/ask/drafts", {
          params: { path: { tripId: id } },
          body: { prompt: prompt.trim() },
        }),
      );
      setDraft(d);
      setPicked(new Set(d.days.flatMap((day) => day.items.map((i) => key(day.day, i.placeId)))));
      if (d.days.every((day) => day.items.length === 0)) setNote(t("ask.nothing"));
    } catch (e) {
      // A used-up limit opens the paywall by itself; only other problems need words here.
      const limit = e instanceof Error && "code" in e && e.code === "limit_reached";
      setNote(limit ? t("ask.limit") : t("ask.error"));
    } finally {
      setBusy(false);
    }
  }

  async function add() {
    if (!draft) return;
    setBusy(true);
    setNote(undefined);
    try {
      const trip = await api.unwrap(
        api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } }),
      );
      const start = trip.dates?.start;
      if (!start) return setNote(t("ask.noDates"));
      const last = trip.dates?.end ?? start;
      const dates = daysInRange(start, last, 366);
      let added = 0;
      for (const day of draft.days) {
        // Days past the end of the trip land on its last day.
        const date = dates[Math.min(day.day, dates.length) - 1] ?? start;
        for (const item of day.items) {
          if (!picked.has(key(day.day, item.placeId))) continue;
          await api.unwrap(
            api.POST("/v1/trips/{tripId}/stops", {
              params: { path: { tripId: id } },
              body: { date, place: { placeId: item.placeId } },
            }),
          );
          added += 1;
        }
      }
      setNote(t("ask.added", { count: added }));
      setPicked(new Set());
    } catch {
      setNote(t("ask.addError"));
    } finally {
      setBusy(false);
    }
  }

  const toggle = (k: string) =>
    setPicked((p) => {
      const n = new Set(p);
      if (!n.delete(k)) n.add(k);
      return n;
    });

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView
        contentContainerStyle={{ padding: 20, gap: 16 }}
        keyboardShouldPersistTaps="handled"
      >
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("ask.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("ask.title")}
          </Text>
        </View>
        <Text>{t("ask.intro")}</Text>
        <Input
          label={t("ask.label")}
          hint={t("ask.hint")}
          value={prompt}
          onChangeText={setPrompt}
          multiline
          maxLength={1000}
        />
        <Button
          label={busy && !draft ? t("ask.working") : t("ask.go")}
          disabled={busy}
          onPress={() => void ask()}
        />
        {note ? <Text accessibilityLiveRegion="polite">{note}</Text> : null}
        {draft && draft.remaining !== undefined && draft.remaining >= 0 ? (
          <Text variant="caption">{t("ask.remaining", { count: draft.remaining })}</Text>
        ) : null}

        {draft?.days.map((day) =>
          day.items.length === 0 ? null : (
            <Card key={day.day} style={{ gap: 8 }}>
              <Text variant="bodyStrong" heading>
                {t("ask.day", { number: day.day })}
              </Text>
              {day.items.map((item) => {
                const k = key(day.day, item.placeId);
                const on = picked.has(k);
                return (
                  <Pressable
                    key={k}
                    accessibilityRole="checkbox"
                    accessibilityLabel={item.name}
                    accessibilityState={{ checked: on }}
                    hitSlop={8}
                    onPress={() => toggle(k)}
                    style={{
                      flexDirection: "row",
                      gap: 12,
                      alignItems: "flex-start",
                      minHeight: 44,
                    }}
                  >
                    <Icon name={on ? "check" : "plus"} color={colors.indigo} />
                    <View style={{ flex: 1 }}>
                      <Text variant="bodyStrong">{item.name}</Text>
                      {item.note ? <Text variant="caption">{item.note}</Text> : null}
                    </View>
                  </Pressable>
                );
              })}
            </Card>
          ),
        )}
        {draft && picked.size > 0 ? (
          <Button
            label={t("ask.add", { count: picked.size })}
            disabled={busy}
            onPress={() => void add()}
          />
        ) : null}
        {draft ? <Text variant="caption">{t("ask.preview")}</Text> : null}
      </ScrollView>
    </SafeAreaView>
  );
}
