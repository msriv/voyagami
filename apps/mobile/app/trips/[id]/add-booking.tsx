import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { daysInRange } from "@/trips/dates";
import { isValidTime } from "@/trips/stops";
import type { Trip } from "@/trips/summary";
import { zonedToUtc } from "@/trips/zoned";
import { Button, Chip, IconButton, Input, Text } from "@/ui";

const kinds = ["flight", "stay", "train", "bus", "activity", "other"] as const;
type Kind = (typeof kinds)[number];

/**
 * Add a booking by hand: what it is, a name, who runs it, and when it starts.
 * shortcut: no end time, places or price, and no file or photo, until the booking detail screen
 * and the file bucket exist.
 */
export default function AddBookingScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [trip, setTrip] = useState<Trip | undefined>();
  const [kind, setKind] = useState<Kind>("flight");
  const [title, setTitle] = useState("");
  const [provider, setProvider] = useState("");
  const [day, setDay] = useState<string | undefined>();
  const [time, setTime] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api
      .unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } }))
      .then(setTrip)
      .catch(() => setError(t("addBooking.errors.generic")));
  }, [api, id, t]);

  const { start, end } = trip?.dates ?? {};
  const days = start && end ? daysInRange(start, end) : [];

  async function save() {
    if (!trip) return;
    if (!title.trim()) return setError(t("addBooking.errors.title"));
    if (!isValidTime(time.trim())) return setError(t("addBooking.errors.time"));
    if (time.trim() && !day) return setError(t("addBooking.errors.day"));
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.POST("/v1/trips/{tripId}/bookings", {
          params: { path: { tripId: id } },
          body: {
            kind,
            title: title.trim(),
            ...(provider.trim() ? { provider: provider.trim() } : null),
            ...(day
              ? {
                  startsAt: zonedToUtc(day, time.trim() || "00:00", trip.timeZone),
                  startTimeZone: trip.timeZone,
                }
              : null),
          },
        }),
      );
      router.back();
    } catch (e) {
      setError(
        t(e instanceof NetworkError ? "addBooking.errors.offline" : "addBooking.errors.generic"),
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView
        contentContainerStyle={{ padding: 20, gap: 16 }}
        keyboardShouldPersistTaps="handled"
      >
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("addBooking.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Text variant="bodyStrong">{"‹"}</Text>
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("addBooking.title")}
          </Text>
        </View>

        <View style={{ gap: 6 }}>
          <Text variant="bodyStrong">{t("addBooking.kindLabel")}</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
            {kinds.map((k) => (
              <Chip
                key={k}
                label={t(`bookings.kinds.${k}`)}
                selected={k === kind}
                onPress={() => setKind(k)}
              />
            ))}
          </View>
        </View>

        <Input
          label={t("addBooking.titleLabel")}
          value={title}
          onChangeText={setTitle}
          maxLength={120}
        />
        <Input
          label={t("addBooking.providerLabel")}
          value={provider}
          onChangeText={setProvider}
          maxLength={120}
        />

        {days.length > 0 ? (
          <View style={{ gap: 6 }}>
            <Text variant="bodyStrong">{t("addBooking.dayLabel")}</Text>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={{ gap: 8 }}
            >
              {days.map((d, i) => (
                <Chip
                  key={d}
                  label={t("addBooking.dayChip", { n: i + 1 })}
                  selected={d === day}
                  onPress={() => setDay(d === day ? undefined : d)}
                />
              ))}
            </ScrollView>
          </View>
        ) : null}
        <Input
          label={t("addBooking.timeLabel")}
          value={time}
          onChangeText={setTime}
          maxLength={5}
          keyboardType="numbers-and-punctuation"
        />

        {error ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}
        <Button label={t("addBooking.save")} loading={busy} onPress={() => void save()} />
      </ScrollView>
    </SafeAreaView>
  );
}
