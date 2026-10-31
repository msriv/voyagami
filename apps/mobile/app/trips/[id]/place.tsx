import { formatTime } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { Image, Linking, ScrollView, Share, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { useTheme } from "@/theme";
import { addCatalogPlace, type CatalogPlaceLike } from "@/trips/addCatalogPlace";
import { Button, Card, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

interface Place extends CatalogPlaceLike {
  id: string;
  photos: { url: string; attribution: string }[];
  sources: { source: string }[];
  openingHours?: { day: (typeof DAYS)[number]; open: string; close: string }[];
}

const DAYS = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"] as const;

type State =
  { kind: "loading" } | { kind: "ready"; place: Place } | { kind: "notFound" } | { kind: "error" };

/** One catalogue place: photos with credits, where the details came from, map, share, add to the trip. */
export default function PlaceScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const { units } = useProfile();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const router = useRouter();
  const { id, placeId, date } = useLocalSearchParams<{
    id: string;
    placeId: string;
    date?: string;
  }>();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [note, setNote] = useState<string | undefined>();

  const load = useCallback(async () => {
    setState({ kind: "loading" });
    try {
      const place = await api.unwrap(
        api.GET("/v1/places/{placeId}", { params: { path: { placeId } } }),
      );
      setState({ kind: "ready", place });
    } catch (e) {
      setState({ kind: e instanceof ApiError && e.code === "not_found" ? "notFound" : "error" });
    }
  }, [api, placeId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function add(p: Place) {
    setNote(undefined);
    try {
      setNote(t((await addCatalogPlace(api, id, p, date)) ? "place.added" : "place.noDay"));
    } catch {
      setNote(t("place.addError"));
    }
  }

  const clock = (hhmm: string) =>
    formatTime(new Date(`1970-01-01T${hhmm}:00Z`), locale, { timeZone: "UTC", clock: units.clock });

  const place = state.kind === "ready" ? state.place : undefined;
  const sourceNames = place ? [...new Set(place.sources.map((s) => s.source))].join(", ") : "";

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("place.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {place?.name ?? ""}
          </Text>
        </View>

        {state.kind === "loading" ? <Text color="muted">{t("place.loading")}</Text> : null}
        {state.kind === "notFound" ? <Text>{t("place.notFound")}</Text> : null}
        {state.kind === "error" ? (
          <Card>
            <Text accessibilityLiveRegion="polite">{t("place.loadError")}</Text>
            <Button label={t("place.retry")} variant="secondary" onPress={() => void load()} />
          </Card>
        ) : null}

        {place ? (
          <>
            {place.photos.map((photo) => (
              <View key={photo.url} style={{ gap: 4 }}>
                <Image
                  source={{ uri: photo.url }}
                  accessibilityIgnoresInvertColors
                  style={{ height: 180, borderRadius: 12, backgroundColor: colors.matcha }}
                />
                <Text variant="caption" color="muted">
                  {t("place.photoBy", { credit: photo.attribution })}
                </Text>
              </View>
            ))}
            {place.address ? <Text>{place.address}</Text> : null}
            {place.openingHours ? (
              <Card style={{ gap: 4 }}>
                <Text variant="bodyStrong" heading>
                  {t("place.hours")}
                </Text>
                {DAYS.map((day) => {
                  const spans = place.openingHours?.filter((p) => p.day === day) ?? [];
                  return (
                    <Text key={day}>
                      {t("place.dayHours", {
                        day: t(`place.days.${day}`),
                        hours: spans.length
                          ? spans
                              .map((p) =>
                                t("place.span", { open: clock(p.open), close: clock(p.close) }),
                              )
                              .join(", ")
                          : t("place.closed"),
                      })}
                    </Text>
                  );
                })}
              </Card>
            ) : null}
            {sourceNames ? (
              <Text variant="caption" color="muted">
                {t("place.sources", { names: sourceNames })}
              </Text>
            ) : null}
            {note ? <Text accessibilityLiveRegion="polite">{note}</Text> : null}
            <Button label={t("place.add")} onPress={() => void add(place)} />
            <Button
              label={t("place.openMap")}
              variant="secondary"
              onPress={() =>
                void Linking.openURL(
                  `https://www.google.com/maps/search/?api=1&query=${place.lat},${place.lng}`,
                )
              }
            />
            <Button
              label={t("place.share")}
              variant="secondary"
              onPress={() =>
                void Share.share({
                  message: [place.name, place.address].filter(Boolean).join(", "),
                })
              }
            />
          </>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}
