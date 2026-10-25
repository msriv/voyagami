import { formatDate, formatTime } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Card, Chip, IconButton, Input, Text } from "@/ui";

type Candidate = components["schemas"]["BookingCandidate"];

/** Below this the model was unsure, so the row asks the person to look twice. */
const SURE = 0.6;

/**
 * Paste a confirmation, review what was found, add the ones that look right.
 * shortcut: pasted text only, no PDF picker, until a document picker is in the app.
 */
export default function ImportBookingScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [text, setText] = useState("");
  const [found, setFound] = useState<Candidate[] | undefined>();
  const [picked, setPicked] = useState<boolean[]>([]);
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  function fail(e: unknown) {
    setError(
      t(
        e instanceof NetworkError
          ? "importBooking.errors.offline"
          : e instanceof ApiError && e.status === 503
            ? "importBooking.errors.unavailable"
            : "importBooking.errors.generic",
      ),
    );
  }

  async function find() {
    if (!text.trim()) return setError(t("importBooking.errors.empty"));
    setBusy(true);
    setError(undefined);
    try {
      const { candidates } = await api.unwrap(
        api.POST("/v1/trips/{tripId}/bookings/import", {
          params: { path: { tripId: id } },
          body: { text: text.trim() },
        }),
      );
      if (candidates.length === 0) setError(t("importBooking.nothing"));
      else {
        setFound(candidates);
        setPicked(candidates.map(() => true));
      }
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function addPicked(list: Candidate[]) {
    setBusy(true);
    setError(undefined);
    // Added ones drop out of the list, so a retry only sends what is left.
    const left: Candidate[] = [];
    let failed: unknown;
    for (const [i, c] of list.entries()) {
      if (!picked[i]) continue;
      try {
        await api.unwrap(
          api.POST("/v1/trips/{tripId}/bookings", {
            params: { path: { tripId: id } },
            body: c.booking,
          }),
        );
      } catch (e) {
        failed = e;
        left.push(c);
      }
    }
    setBusy(false);
    if (!failed) return router.back();
    setFound(left);
    setPicked(left.map(() => true));
    if (failed instanceof NetworkError) fail(failed);
    else setError(t("importBooking.errors.partial"));
  }

  function when(c: Candidate): string {
    const { startsAt, startTimeZone } = c.booking;
    if (!startsAt) return t("importBooking.noTime");
    const at = new Date(startsAt);
    const timeZone = startTimeZone ?? "UTC";
    return `${formatDate(at, locale, { timeZone })}, ${formatTime(at, locale, { timeZone })}`;
  }

  const count = picked.filter(Boolean).length;

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView
        contentContainerStyle={{ padding: 20, gap: 16 }}
        keyboardShouldPersistTaps="handled"
      >
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("importBooking.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Text variant="bodyStrong">{"‹"}</Text>
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {found ? t("importBooking.reviewTitle") : t("importBooking.title")}
          </Text>
        </View>

        {found ? (
          <>
            <Text color="muted">{t("importBooking.reviewHint")}</Text>
            {found.map((c, i) => (
              <Card key={`${c.booking.title}-${i}`}>
                <View style={{ gap: 2 }}>
                  <Text variant="caption">
                    {t(`bookings.kinds.${c.booking.kind}`).toUpperCase()}
                  </Text>
                  <Text variant="bodyStrong">{c.booking.title}</Text>
                  {c.booking.provider ? <Text color="muted">{c.booking.provider}</Text> : null}
                  <Text variant="caption">{when(c)}</Text>
                  {c.confidence < SURE ? (
                    <Text variant="caption" color="vermilion">
                      {t("importBooking.checkThis")}
                    </Text>
                  ) : null}
                  <View style={{ flexDirection: "row", paddingTop: 8 }}>
                    <Chip
                      label={t("importBooking.include", { name: c.booking.title })}
                      selected={picked[i] ?? false}
                      onPress={() => setPicked(picked.map((p, j) => (j === i ? !p : p)))}
                    />
                  </View>
                </View>
              </Card>
            ))}
            {error ? (
              <Text color="vermilion" accessibilityLiveRegion="polite">
                {error}
              </Text>
            ) : null}
            <Button
              label={t("importBooking.addSelected", { count })}
              loading={busy}
              disabled={count === 0}
              onPress={() => void addPicked(found)}
            />
            <Button
              label={t("importBooking.startOver")}
              variant="secondary"
              onPress={() => {
                setFound(undefined);
                setError(undefined);
              }}
            />
          </>
        ) : (
          <>
            <Text color="muted">{t("importBooking.intro")}</Text>
            <Input
              label={t("importBooking.textLabel")}
              value={text}
              onChangeText={setText}
              multiline
              maxLength={20000}
            />
            {error ? (
              <Text color="vermilion" accessibilityLiveRegion="polite">
                {error}
              </Text>
            ) : null}
            <Button label={t("importBooking.find")} loading={busy} onPress={() => void find()} />
          </>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}
