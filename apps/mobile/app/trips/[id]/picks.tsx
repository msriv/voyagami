import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { Image, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { usePaywall } from "@/plans/Paywall";
import { useTripPlan } from "@/plans/useTripPlan";
import { useTheme } from "@/theme";
import { addCatalogPlace } from "@/trips/addCatalogPlace";
import { Button, Card, Chip, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

const GROUPS = ["see", "eat", "stay", "do", "shop", "nightlife"] as const;
type Group = (typeof GROUPS)[number];
type Reason = "style" | "quality" | "distance" | "popularity";

interface Pick {
  place: {
    id: string;
    name: string;
    lat: number;
    lng: number;
    address?: string;
    group: Group;
    photos: { url: string; attribution: string }[];
  };
  reasons: Reason[];
}

type State =
  | { kind: "loading" }
  | { kind: "ready"; items: Pick[]; hasMore: boolean }
  | { kind: "noPosition" }
  | { kind: "error" };

/**
 * Picks for you: places ranked for the crew, each with plain reasons and its photo credit.
 * Free shows the first few; "See more" asks the paywall, and the server decides what is allowed.
 */
export default function PicksScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const paywall = usePaywall();
  const { plan } = useTripPlan(useLocalSearchParams<{ id: string }>().id);
  const { id, date } = useLocalSearchParams<{ id: string; date?: string }>();
  const [group, setGroup] = useState<Group>("eat");
  const [limit, setLimit] = useState<number | undefined>();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [note, setNote] = useState<string | undefined>();

  const load = useCallback(async () => {
    setState({ kind: "loading" });
    try {
      const r = await api.unwrap(
        api.GET("/v1/trips/{tripId}/recommendations", {
          params: {
            path: { tripId: id },
            query: { category: group, ...(limit ? { limit } : null) },
          },
        }),
      );
      setState({ kind: "ready", items: r.items, hasMore: r.hasMore });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "bad_request"
          ? { kind: "noPosition" }
          : { kind: "error" },
      );
    }
  }, [api, id, group, limit]);

  useEffect(() => {
    void load();
  }, [load]);

  async function add(p: Pick) {
    setNote(undefined);
    try {
      const ok = await addCatalogPlace(api, id, p.place, date);
      setNote(ok ? t("picks.added", { name: p.place.name }) : t("picks.noDay"));
    } catch {
      setNote(t("picks.addError"));
    }
  }

  const seeMore = () => {
    if (plan === "free") paywall.show("topPicks");
    else setLimit(50);
  };

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("picks.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("picks.title")}
          </Text>
        </View>

        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
          {GROUPS.map((g) => (
            <Chip
              key={g}
              label={t(`picks.groups.${g}`)}
              selected={g === group}
              onPress={() => setGroup(g)}
            />
          ))}
        </View>

        {note ? <Text accessibilityLiveRegion="polite">{note}</Text> : null}
        {state.kind === "loading" ? <Text color="muted">{t("picks.loading")}</Text> : null}
        {state.kind === "noPosition" ? (
          <Text accessibilityLiveRegion="polite">{t("picks.noPosition")}</Text>
        ) : null}
        {state.kind === "error" ? (
          <Card>
            <Text accessibilityLiveRegion="polite">{t("picks.loadError")}</Text>
            <Button label={t("picks.retry")} variant="secondary" onPress={() => void load()} />
          </Card>
        ) : null}
        {state.kind === "ready" && state.items.length === 0 ? (
          <Text>{t("picks.empty")}</Text>
        ) : null}
        {state.kind === "ready"
          ? state.items.map((p) => {
              const photo = p.place.photos[0];
              return (
                <Card key={p.place.id} style={{ gap: 8 }}>
                  {photo ? (
                    <View style={{ gap: 4 }}>
                      <Image
                        source={{ uri: photo.url }}
                        accessibilityIgnoresInvertColors
                        style={{ height: 140, borderRadius: 12, backgroundColor: colors.matcha }}
                      />
                      <Text variant="caption" color="muted">
                        {t("picks.photoBy", { credit: photo.attribution })}
                      </Text>
                    </View>
                  ) : null}
                  <Text variant="bodyStrong" heading>
                    {p.place.name}
                  </Text>
                  {p.place.address ? (
                    <Text variant="caption" color="muted">
                      {p.place.address}
                    </Text>
                  ) : null}
                  {p.reasons.map((r) => (
                    <Text key={r}>{t(`picks.reasons.${r}`)}</Text>
                  ))}
                  <Button
                    label={t("picks.details", { name: p.place.name })}
                    variant="ghost"
                    onPress={() => router.push(`/trips/${id}/place?placeId=${p.place.id}`)}
                  />
                  <Button
                    label={t("picks.add", { name: p.place.name })}
                    variant="secondary"
                    onPress={() => void add(p)}
                  />
                </Card>
              );
            })
          : null}
        {state.kind === "ready" && state.hasMore ? (
          <Button label={t("picks.seeMore")} variant="secondary" onPress={seeMore} />
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}
