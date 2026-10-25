import { formatDate } from "@voyagami/locale";
import { useRouter } from "expo-router";
import { Pressable, View } from "react-native";

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
  const stripe = [colors.vermilion, colors.washi, colors.indigo, colors.washi];
  const country = trip.destinations[0]?.countryCode;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={open}
      style={{
        backgroundColor: colors.indigo,
        borderRadius: radius.card + 4,
        padding: 22,
        paddingBottom: 28,
        gap: 18,
        overflow: "hidden",
      }}
    >
      <WavePattern opacity={0.09} />
      <View style={{ flexDirection: "row", justifyContent: "space-between", gap: 12 }}>
        <View style={{ flex: 1, gap: 8 }}>
          <View
            style={{
              alignSelf: "flex-start",
              backgroundColor: "rgba(255,255,255,0.18)",
              borderRadius: radius.pill,
              paddingHorizontal: 11,
              paddingVertical: 5,
            }}
          >
            <Text variant="caption" color="white" style={{ fontWeight: "700", letterSpacing: 0.5 }}>
              {t("home.upNext").toUpperCase()}
            </Text>
          </View>
          <Text variant="display" color="white" style={{ fontSize: 28, lineHeight: 30 }}>
            {trip.title}
          </Text>
          <Text color="white" style={{ fontSize: 14 }}>
            {[places, dates].filter(Boolean).join(t("home.listSeparator"))}
          </Text>
        </View>
        {country ? (
          <View
            accessibilityElementsHidden
            importantForAccessibility="no-hide-descendants"
            style={{
              width: 74,
              height: 86,
              backgroundColor: colors.white,
              borderRadius: 6,
              borderWidth: 3,
              borderColor: colors.vermilion,
              borderStyle: "dashed",
              transform: [{ rotate: "6deg" }],
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Text variant="display" style={{ fontSize: 26, lineHeight: 30 }}>
              {country}
            </Text>
          </View>
        ) : null}
      </View>
      {when ? (
        <View
          style={{
            borderTopWidth: 2,
            borderTopColor: "rgba(255,255,255,0.35)",
            borderStyle: "dashed",
            paddingTop: 14,
            flexDirection: "row",
            alignItems: "baseline",
            gap: 8,
          }}
        >
          {c.kind === "in" && c.days > 0 ? (
            <>
              <Text variant="display" color="white" style={{ fontSize: 44, lineHeight: 46 }}>
                {c.days}
              </Text>
              <Text color="white" style={{ fontWeight: "700" }}>
                {t("home.daysToGo", { count: c.days })}
              </Text>
            </>
          ) : (
            <Text variant="title" color="white">
              {when}
            </Text>
          )}
        </View>
      ) : null}
      <View
        accessibilityElementsHidden
        importantForAccessibility="no-hide-descendants"
        style={{
          position: "absolute",
          left: 0,
          right: 0,
          bottom: 0,
          height: 6,
          flexDirection: "row",
        }}
      >
        {Array.from({ length: 40 }, (_, i) => (
          <View
            key={i}
            style={{
              flex: 1,
              backgroundColor: stripe[i % 4] ?? colors.washi,
            }}
          />
        ))}
      </View>
    </Pressable>
  );
}
