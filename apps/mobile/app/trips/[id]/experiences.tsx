import { formatDistance } from "@voyagami/locale";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { Image, Linking, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { useTheme } from "@/theme";
import { isValidTime } from "@/trips/stops";
import { Button, Card, IconButton, Input, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Experience = components["schemas"]["Experience"];
type Link = Experience["links"][number];
type State =
  | { kind: "loading" }
  | { kind: "ready"; items: Experience[] }
  | { kind: "off" }
  | { kind: "error" };

/**
 * Things to do near the trip. Partner links leave the app (nothing is booked here); "Add to itinerary"
 * puts the place on the trip's first day at the time the person types.
 */
export default function ExperiencesScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const { locale, units } = useProfile();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [adding, setAdding] = useState<string | undefined>();
  const [time, setTime] = useState("");
  const [note, setNote] = useState<string | undefined>();

  const load = useCallback(async () => {
    setState({ kind: "loading" });
    try {
      const { items } = await api.unwrap(
        api.GET("/v1/trips/{tripId}/experiences", { params: { path: { tripId: id } } }),
      );
      setState({ kind: "ready", items });
    } catch (e) {
      // 503 means the list is switched off on the server.
      setState(e instanceof ApiError && e.status === 503 ? { kind: "off" } : { kind: "error" });
    }
  }, [api, id]);

  useEffect(() => {
    void load();
  }, [load]);

  const open = (link: Link) => {
    // Counting the tap is best effort: it must never stop the person leaving for the partner.
    void api
      .POST("/v1/trips/{tripId}/experiences/clicks", {
        params: { path: { tripId: id } },
        body: { partnerId: link.partnerId },
      })
      .catch(() => undefined);
    void Linking.openURL(link.url);
  };

  async function add(item: Experience) {
    if (!isValidTime(time.trim())) return setNote(t("experiences.timeError"));
    setNote(undefined);
    try {
      const trip = await api.unwrap(
        api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } }),
      );
      const date = trip.dates?.start;
      if (!date) return setNote(t("experiences.noDates"));
      await api.unwrap(
        api.POST("/v1/trips/{tripId}/stops", {
          params: { path: { tripId: id } },
          body: {
            date,
            place: { placeId: item.placeId },
            ...(time.trim() ? { startLocal: time.trim() } : null),
          },
        }),
      );
      setNote(t("experiences.added", { name: item.name }));
      setAdding(undefined);
      setTime("");
    } catch {
      setNote(t("experiences.addError"));
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
            accessibilityLabel={t("experiences.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("experiences.title")}
          </Text>
        </View>
        <Text>{t("experiences.note")}</Text>

        {note ? <Text accessibilityLiveRegion="polite">{note}</Text> : null}
        {state.kind === "loading" ? <Text color="muted">{t("experiences.loading")}</Text> : null}
        {state.kind === "off" ? (
          <Text accessibilityLiveRegion="polite">{t("experiences.off")}</Text>
        ) : null}
        {state.kind === "error" ? (
          <Card>
            <Text accessibilityLiveRegion="polite">{t("experiences.loadError")}</Text>
            <Button
              label={t("experiences.retry")}
              variant="secondary"
              onPress={() => void load()}
            />
          </Card>
        ) : null}
        {state.kind === "ready" && state.items.length === 0 ? (
          <Text accessibilityLiveRegion="polite">{t("experiences.empty")}</Text>
        ) : null}
        {state.kind === "ready"
          ? state.items.map((item) => (
              <Card key={item.placeId} style={{ gap: 8 }}>
                {item.photo ? (
                  <View style={{ gap: 4 }}>
                    <Image
                      source={{ uri: item.photo.url }}
                      accessibilityIgnoresInvertColors
                      style={{ height: 140, borderRadius: 12, backgroundColor: colors.matcha }}
                    />
                    <Text variant="caption" color="muted">
                      {t("experiences.photoBy", { credit: item.photo.attribution })}
                    </Text>
                  </View>
                ) : null}
                <Text variant="bodyStrong" heading>
                  {item.name}
                </Text>
                <Text variant="caption" color="muted">
                  {t("experiences.away", {
                    distance: formatDistance(item.distanceMeters, units.distance, locale),
                  })}
                </Text>
                {item.links.map((link) => (
                  <Button
                    key={link.partnerId}
                    label={t("experiences.openPartner", { name: link.name })}
                    onPress={() => open(link)}
                  />
                ))}
                {adding === item.placeId ? (
                  <>
                    <Input
                      label={t("experiences.timeLabel")}
                      value={time}
                      onChangeText={setTime}
                      keyboardType="numbers-and-punctuation"
                    />
                    <Button label={t("experiences.confirm")} onPress={() => void add(item)} />
                  </>
                ) : (
                  <Button
                    label={t("experiences.add", { name: item.name })}
                    variant="secondary"
                    onPress={() => {
                      setNote(undefined);
                      setAdding(item.placeId);
                    }}
                  />
                )}
              </Card>
            ))
          : null}
      </ScrollView>
    </SafeAreaView>
  );
}
