import { formatDate } from "@voyagami/locale";
import { useRouter } from "expo-router";
import { Pressable } from "react-native";

import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Card, Text, WavePattern } from "@/ui";

import { isoToDate } from "./dates";
import { countdown, type Trip } from "./summary";

export interface TripCardProps {
  trip: Trip;
  locale: string;
  today: string;
  /** The next trip gets the big indigo card. */
  featured?: boolean;
}

export function TripCard({ trip, locale, today, featured = false }: TripCardProps) {
  const { t } = useTranslation();
  const router = useRouter();
  const { colors, radius } = useTheme();
  const { start, end, flexible } = trip.dates;
  const dates =
    start && end
      ? t("home.dateRange", {
          start: formatDate(isoToDate(start), locale, { timeZone: "UTC" }),
          end: formatDate(isoToDate(end), locale, { timeZone: "UTC", withYear: true }),
        })
      : t("home.flexibleDays", { count: flexible?.days ?? 0 });
  const places = trip.destinations.map((d) => d.name).join(t("home.listSeparator"));
  const c = countdown(trip, today);
  const when =
    c.kind === "ongoing"
      ? t("home.ongoing")
      : c.kind === "in"
        ? c.days === 0
          ? t("home.startsToday")
          : t("home.inDays", { count: c.days })
        : undefined;
  const label = [trip.title, places, dates, when].filter(Boolean).join(". ");

  const open = () => router.push(`/trips/${trip.id}`);
  if (!featured) {
    return (
      <Card>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={label}
          onPress={open}
          style={{ gap: 2 }}
        >
          <Text variant="title">{trip.title}</Text>
          <Text color="muted">{places}</Text>
          <Text variant="caption">
            {[dates, when].filter(Boolean).join(t("home.listSeparator"))}
          </Text>
        </Pressable>
      </Card>
    );
  }
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={open}
      style={{
        backgroundColor: colors.indigo,
        borderRadius: radius.card + 4,
        padding: 22,
        gap: 10,
        overflow: "hidden",
      }}
    >
      <WavePattern opacity={0.09} />
      {when ? (
        <Text variant="caption" color="white" style={{ fontWeight: "800" }}>
          {when.toUpperCase()}
        </Text>
      ) : null}
      <Text variant="display" color="white">
        {trip.title}
      </Text>
      <Text color="white">{places}</Text>
      <Text variant="caption" color="white">
        {dates}
      </Text>
    </Pressable>
  );
}
