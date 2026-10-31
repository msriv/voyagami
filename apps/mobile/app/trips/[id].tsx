import { formatDate } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { useTheme } from "@/theme";
import { useTripChanges } from "@/trips/useTripChanges";
import { countryName } from "@/trips/destinations";
import { daysBetween, daysInRange, isoToDate } from "@/trips/dates";
import type { Trip } from "@/trips/summary";
import { TripTasks } from "@/trips/TripTasks";
import { SyncStatus } from "@/local/SyncStatus";
import { Button, Card, IconButton, Input, Sheet, Text, WavePattern } from "@/ui";
import { Icon } from "@/ui/icons";

type State =
  { kind: "loading" } | { kind: "ready"; trip: Trip } | { kind: "notFound" } | { kind: "error" };

export default function TripScreen() {
  const { t } = useTranslation();
  const { colors, radius } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const { region } = useProfile();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [crew, setCrew] = useState<string[]>([]);
  const [menu, setMenu] = useState(false);
  const [name, setName] = useState("");
  const [formError, setFormError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const trip = await api.unwrap(
        api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } }),
      );
      setState({ kind: "ready", trip });
      setName(trip.title);
      // The avatar stack is decoration: a failure here must not block the trip.
      api
        .unwrap(api.GET("/v1/trips/{tripId}/members", { params: { path: { tripId: id } } }))
        .then((r) => setCrew(r.members.map((m) => m.displayName ?? "")))
        .catch(() => undefined);
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  useTripChanges(id, () => void load());

  useEffect(() => {
    void load();
  }, [load]);

  async function change(body: { title?: string; status?: "past" | "planned" }) {
    setBusy(true);
    setFormError(undefined);
    try {
      const trip = await api.unwrap(
        api.PATCH("/v1/trips/{tripId}", { params: { path: { tripId: id } }, body }),
      );
      setState({ kind: "ready", trip });
      setMenu(false);
    } catch (e) {
      setFormError(t(e instanceof NetworkError ? "trip.errors.offline" : "trip.errors.generic"));
    } finally {
      setBusy(false);
    }
  }

  const back = (
    <IconButton
      accessibilityLabel={t("trip.back")}
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
          <Card>
            <Text accessibilityLiveRegion="polite">
              {state.kind === "notFound" ? t("trip.notFound") : t("trip.loadError")}
            </Text>
            {state.kind === "error" ? (
              <Button label={t("trip.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </Card>
        )}
      </SafeAreaView>
    );
  }

  const { trip } = state;
  const { start, end, flexible } = trip.dates;
  const dates =
    start && end
      ? t("trip.dateRange", {
          start: formatDate(isoToDate(start), locale, { timeZone: "UTC" }),
          end: formatDate(isoToDate(end), locale, { timeZone: "UTC", withYear: true }),
        }) +
        ", " +
        t("trip.days", { count: daysBetween(start, end) })
      : t("trip.flexibleDays", { count: flexible?.days ?? 0 });
  const countries = [...new Set(trip.destinations.map((d) => d.countryCode))].map(countryName);
  const abroad = trip.destinations.some((d) => region && d.countryCode !== region);
  const kicker = [...countries, ...(abroad ? [t("trip.international")] : [])]
    .join(" · ")
    .toUpperCase();
  const archived = trip.status === "past";
  const canEdit = trip.role !== "viewer";
  const sections = ["itinerary", "map", "bookings", "money", "journal", "crew", "recap"] as const;

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.indigo }}>
      <ScrollView style={{ backgroundColor: colors.washi }} contentContainerStyle={{ flexGrow: 1 }}>
        <View
          style={{
            backgroundColor: colors.indigo,
            padding: 20,
            gap: 12,
            borderBottomLeftRadius: 32,
            borderBottomRightRadius: 32,
            overflow: "hidden",
          }}
        >
          <WavePattern opacity={0.09} />
          <View style={{ flexDirection: "row", justifyContent: "space-between" }}>
            <GlassButton label={t("trip.back")} icon="back" onPress={() => router.back()} />
            <GlassButton label={t("trip.menu")} icon="more" onPress={() => setMenu(true)} />
          </View>
          <View style={{ gap: 6 }}>
            {archived ? (
              <Text variant="caption" color="white" style={{ fontWeight: "800" }}>
                {t("trip.archived").toUpperCase()}
              </Text>
            ) : null}
            {kicker ? (
              <Text
                variant="caption"
                color="white"
                style={{ fontWeight: "800", letterSpacing: 1.5 }}
              >
                {kicker}
              </Text>
            ) : null}
            <Text variant="display" color="white" heading style={{ fontSize: 34, lineHeight: 36 }}>
              {trip.title}
            </Text>
            <Text color="white">
              {trip.destinations.map((d) => d.name).join(t("home.listSeparator"))}
            </Text>
            <Text variant="caption" color="white">
              {dates}
            </Text>
            <SyncStatus onDark />
            <Text variant="caption" color="white">
              {t("trip.roleLine", { role: t(`trip.roles.${trip.role}`) })}
            </Text>
          </View>
          <View
            style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}
          >
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={t("trip.crewLabel", { names: crew.filter(Boolean).join(", ") })}
              onPress={() => router.push(`/trips/${trip.id}/crew`)}
              style={{ flexDirection: "row", minHeight: 44, alignItems: "center" }}
            >
              {crew.slice(0, 4).map((n, i) => (
                <View
                  key={i}
                  style={{
                    width: 34,
                    height: 34,
                    borderRadius: 17,
                    borderWidth: 2,
                    borderColor: colors.indigo,
                    marginLeft: i === 0 ? 0 : -10,
                    alignItems: "center",
                    justifyContent: "center",
                    backgroundColor: [colors.vermilion, colors.gold, colors.matcha, colors.white][
                      i
                    ],
                  }}
                >
                  <Text variant="caption" color="ink" style={{ fontWeight: "800" }}>
                    {n.slice(0, 1).toUpperCase()}
                  </Text>
                </View>
              ))}
            </Pressable>
            {trip.role === "owner" ? (
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={t("trip.invite")}
                onPress={() => router.push(`/trips/${trip.id}/crew`)}
                style={{
                  minHeight: 44,
                  paddingHorizontal: 16,
                  borderRadius: 99,
                  justifyContent: "center",
                  backgroundColor: colors.white,
                }}
              >
                <Text variant="bodyStrong">{t("trip.invite")}</Text>
              </Pressable>
            ) : null}
          </View>
        </View>

        <View
          style={{
            flexDirection: "row",
            flexWrap: "wrap",
            gap: 6,
            paddingHorizontal: 20,
            paddingTop: 14,
          }}
        >
          {sections.map((key) => (
            <Pressable
              key={key}
              accessibilityRole="button"
              accessibilityLabel={t(`trip.${key}`)}
              onPress={() => router.push(`/trips/${trip.id}/${key}`)}
              style={{
                minHeight: 44,
                paddingHorizontal: 16,
                borderRadius: 99,
                justifyContent: "center",
                backgroundColor: colors.white,
              }}
            >
              <Text variant="bodyStrong">{t(`trip.${key}`)}</Text>
            </Pressable>
          ))}
        </View>

        <View style={{ padding: 20, gap: 18 }}>
          {start && end ? (
            <View style={{ gap: 8 }}>
              <Text variant="title" heading>
                {t("trip.dayStrip")}
              </Text>
              <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={{ gap: 8 }}
              >
                {daysInRange(start, end).map((iso, i) => (
                  <View
                    key={iso}
                    accessible
                    accessibilityLabel={`${t("trip.dayNumber", { n: i + 1 })}, ${formatDate(isoToDate(iso), locale, { timeZone: "UTC" })}`}
                    style={{
                      backgroundColor: colors.white,
                      borderRadius: radius.button,
                      padding: 12,
                      minWidth: 72,
                      alignItems: "center",
                    }}
                  >
                    <Text variant="caption">{t("trip.dayNumber", { n: i + 1 })}</Text>
                    <Text variant="bodyStrong">
                      {formatDate(isoToDate(iso), locale, { timeZone: "UTC" })}
                    </Text>
                  </View>
                ))}
              </ScrollView>
            </View>
          ) : null}

          {archived ? null : <TripTasks tripId={trip.id} canEdit={canEdit} />}
        </View>
      </ScrollView>

      <Sheet
        visible={menu}
        onClose={() => setMenu(false)}
        title={t("trip.menu")}
        closeLabel={t("trip.menuClose")}
        closeIcon={<Icon name="close" color={colors.ink} />}
      >
        {canEdit ? (
          <View style={{ gap: 12, paddingBottom: 12 }}>
            <Input label={t("trip.nameLabel")} value={name} onChangeText={setName} maxLength={80} />
            <Button
              label={t("trip.save")}
              loading={busy}
              onPress={() =>
                name.trim()
                  ? void change({ title: name.trim() })
                  : setFormError(t("trip.errors.blankName"))
              }
            />
            {trip.role === "owner" ? (
              <>
                <Button
                  label={archived ? t("trip.unarchive") : t("trip.archive")}
                  variant="secondary"
                  loading={busy}
                  onPress={() => void change({ status: archived ? "planned" : "past" })}
                />
                {archived ? null : <Text variant="caption">{t("trip.archiveHint")}</Text>}
              </>
            ) : null}
            {formError ? (
              <Text color="vermilion" accessibilityLiveRegion="polite">
                {formError}
              </Text>
            ) : null}
          </View>
        ) : (
          <Text>{t("trip.viewerNote")}</Text>
        )}
      </Sheet>
    </SafeAreaView>
  );
}

function GlassButton({
  label,
  icon,
  onPress,
}: {
  label: string;
  icon: "back" | "more";
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={{
        width: 44,
        height: 44,
        borderRadius: 22,
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: "rgba(255,255,255,0.18)",
      }}
    >
      <Icon name={icon} color="#FFFFFF" />
    </Pressable>
  );
}
