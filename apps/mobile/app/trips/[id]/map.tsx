import { formatDate } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useMemo, useState } from "react";
import { ScrollView, View } from "react-native";
import MapView, { Marker, Polyline } from "react-native-maps";
import { SafeAreaView } from "react-native-safe-area-context";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { isoToDate } from "@/trips/dates";
import { daysWithStops, mapPins } from "@/trips/mapPins";
import type { Stop } from "@/trips/stops";
import { Card, Chip, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

/** Every stop with a pin on a map, numbered by day order, with a day filter and a list for screen readers. */
export default function MapScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [stops, setStops] = useState<Stop[] | undefined>();
  const [failed, setFailed] = useState(false);
  const [day, setDay] = useState<string | undefined>();

  useEffect(() => {
    api
      .unwrap(api.GET("/v1/trips/{tripId}/stops", { params: { path: { tripId: id } } }))
      .then((r) => setStops(r.stops))
      .catch(() => setFailed(true));
  }, [api, id]);

  const days = useMemo(() => daysWithStops(stops ?? []), [stops]);
  const shown = day ? [day] : days;
  const pins = useMemo(() => mapPins(stops ?? [], shown), [stops, shown]);
  const dayLabel = (iso: string) => formatDate(isoToDate(iso), locale, { timeZone: "UTC" });

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <View style={{ flexDirection: "row", alignItems: "center", gap: 12, padding: 20 }}>
        <IconButton
          accessibilityLabel={t("map.back")}
          variant="secondary"
          onPress={() => router.back()}
        >
          <Icon name="back" color={colors.ink} />
        </IconButton>
        <Text variant="title" heading style={{ flex: 1 }}>
          {t("map.title")}
        </Text>
      </View>

      {failed ? (
        <Card style={{ margin: 20 }}>
          <Text accessibilityLiveRegion="polite">{t("map.error")}</Text>
        </Card>
      ) : !stops ? (
        <Text style={{ padding: 20 }}>{t("map.loading")}</Text>
      ) : days.length === 0 ? (
        <Card style={{ margin: 20 }}>
          <Text>{t("map.empty")}</Text>
        </Card>
      ) : (
        <>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={{ gap: 8, paddingHorizontal: 20, paddingBottom: 12 }}
            style={{ flexGrow: 0 }}
          >
            <Chip label={t("map.allDays")} selected={!day} onPress={() => setDay(undefined)} />
            {days.map((d) => (
              <Chip
                key={d}
                label={dayLabel(d)}
                selected={d === day}
                onPress={() => setDay(d === day ? undefined : d)}
              />
            ))}
          </ScrollView>

          <View
            style={{ height: 280 }}
            accessibilityElementsHidden
            importantForAccessibility="no-hide-descendants"
          >
            <MapView
              key={shown.join(",")}
              style={{ flex: 1 }}
              {...(pins[0]
                ? {
                    initialRegion: {
                      latitude: pins[0].lat,
                      longitude: pins[0].lng,
                      latitudeDelta: 0.1,
                      longitudeDelta: 0.1,
                    },
                  }
                : null)}
              ref={(m) =>
                m?.fitToCoordinates(
                  pins.map((p) => ({ latitude: p.lat, longitude: p.lng })),
                  { edgePadding: { top: 40, right: 40, bottom: 40, left: 40 }, animated: false },
                )
              }
            >
              {shown.map((d) => {
                const line = pins.filter((p) => p.date === d);
                return line.length > 1 ? (
                  <Polyline
                    key={d}
                    coordinates={line.map((p) => ({ latitude: p.lat, longitude: p.lng }))}
                    strokeColor={colors.indigo}
                    strokeWidth={3}
                  />
                ) : null;
              })}
              {pins.map((p) => (
                <Marker
                  key={p.id}
                  coordinate={{ latitude: p.lat, longitude: p.lng }}
                  title={`${p.number}. ${p.name}`}
                >
                  <View
                    style={{
                      width: 30,
                      height: 30,
                      borderRadius: 15,
                      backgroundColor: colors.vermilion,
                      borderWidth: 2,
                      borderColor: colors.white,
                      alignItems: "center",
                      justifyContent: "center",
                    }}
                  >
                    <Text variant="caption" color="white" style={{ fontWeight: "800" }}>
                      {p.number}
                    </Text>
                  </View>
                </Marker>
              ))}
            </MapView>
          </View>

          <ScrollView contentContainerStyle={{ padding: 20, gap: 8 }}>
            <Text variant="bodyStrong">{t("map.listHeading")}</Text>
            {pins.length === 0 ? <Text color="muted">{t("map.noPins")}</Text> : null}
            {pins.map((p) => (
              <Text
                key={p.id}
                accessibilityLabel={t("map.pinLabel", {
                  n: p.number,
                  name: p.name,
                  day: dayLabel(p.date),
                })}
              >
                {`${p.number}. ${p.name}`}
              </Text>
            ))}
          </ScrollView>
        </>
      )}
    </SafeAreaView>
  );
}
