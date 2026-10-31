import { formatDate, formatDistance, formatMoney } from "@voyagami/locale";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { Pressable, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useProfile } from "@/profile";
import { useTheme } from "@/theme";
import { isoToDate } from "@/trips/dates";
import { recapCards, type RecapCard } from "@/trips/recapCards";
import { Button, IconButton, Text, useReducedMotion } from "@/ui";
import { Icon } from "@/ui/icons";

type Recap = components["schemas"]["Recap"];
type Trip = components["schemas"]["Trip"];

type State =
  | { kind: "loading" }
  | { kind: "ready"; recap: Recap; trip: Trip; names: Record<string, string> }
  | { kind: "notFound" }
  | { kind: "error" };

/** How long each card stays up when nobody is holding or pausing. */
const CARD_MS = 5000;

/**
 * The recap as a story: cards that move on by themselves, a tap to go forward or back, a hold to
 * pause. With reduce motion on, nothing moves on its own. Visible buttons do the same jobs for
 * people who can't tap and hold.
 */
export default function RecapPlayerScreen() {
  const { t } = useTranslation();
  const { colors, radius } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const reduceMotion = useReducedMotion();
  const { locale, units } = useProfile();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  const [held, setHeld] = useState(false);

  useEffect(() => {
    let live = true;
    (async () => {
      try {
        const path = { params: { path: { tripId: id } } };
        const [recap, trip, { members }] = await Promise.all([
          api.unwrap(api.GET("/v1/trips/{tripId}/recap", path)),
          api.unwrap(api.GET("/v1/trips/{tripId}", path)),
          api.unwrap(api.GET("/v1/trips/{tripId}/members", path)),
        ]);
        const names = Object.fromEntries(members.map((m) => [m.uid, m.displayName ?? ""]));
        if (live) setState({ kind: "ready", recap, trip, names });
      } catch (e) {
        if (live) {
          setState(
            e instanceof ApiError && e.code === "not_found"
              ? { kind: "notFound" }
              : { kind: "error" },
          );
        }
      }
    })();
    return () => {
      live = false;
    };
  }, [api, id]);

  const cards = state.kind === "ready" ? recapCards(state.recap) : [];
  const last = cards.length - 1;
  const playing = !reduceMotion && !paused && !held && index < last;

  useEffect(() => {
    if (!playing) return;
    const timer = setTimeout(() => setIndex((i) => Math.min(i + 1, last)), CARD_MS);
    return () => clearTimeout(timer);
  }, [playing, index, last]);

  const go = useCallback(
    (delta: number) => setIndex((i) => Math.max(0, Math.min(i + delta, last))),
    [last],
  );

  const close = (
    <IconButton
      accessibilityLabel={t("recapPlayer.close")}
      variant="secondary"
      onPress={() => router.back()}
    >
      <Icon name="close" color={colors.ink} />
    </IconButton>
  );

  if (state.kind !== "ready") {
    return (
      <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi, padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row" }}>{close}</View>
        {state.kind === "loading" ? null : (
          <Text>{state.kind === "notFound" ? t("recap.notFound") : t("recap.loadError")}</Text>
        )}
      </SafeAreaView>
    );
  }

  const { recap, trip, names } = state;
  const day = (iso: string) => formatDate(isoToDate(iso), locale, { timeZone: "UTC" });
  const money = (n: number) => formatMoney(n, recap.currency, locale);
  const card: RecapCard = cards[index] ?? { kind: "cover" };

  const body = (() => {
    switch (card.kind) {
      case "cover": {
        const { start, end } = trip.dates;
        return (
          <>
            <Text variant="caption">{t("recap.stamped")}</Text>
            <Text variant="display" heading>
              {trip.title}
            </Text>
            {start && end ? (
              <Text color="muted">
                {t("trip.dateRange", {
                  start: day(start),
                  end: formatDate(isoToDate(end), locale, { timeZone: "UTC", withYear: true }),
                })}
              </Text>
            ) : null}
          </>
        );
      }
      case "stats":
        return (
          <>
            <Text variant="display" heading>
              {t("recap.days", { count: recap.days })}
            </Text>
            <Text variant="title">{t("recap.stops", { count: recap.stops })}</Text>
            <Text variant="title">{t("recap.friends", { count: recap.crew })}</Text>
            {recap.distanceMeters > 0 ? (
              <Text color="muted">
                {t("recap.distance", {
                  distance: formatDistance(recap.distanceMeters, units.distance, locale),
                })}
              </Text>
            ) : null}
          </>
        );
      case "money":
        return (
          <>
            <Text variant="display" heading>
              {t("recap.spent", { amount: money(recap.spentMinor) })}
            </Text>
            {recap.remainingMinor != null ? (
              <Text variant="title">
                {t(recap.remainingMinor >= 0 ? "recap.underBudget" : "recap.overBudget", {
                  amount: money(Math.abs(recap.remainingMinor)),
                })}
              </Text>
            ) : null}
            {recap.settledUp != null ? (
              <Text color="muted">{t(recap.settledUp ? "recap.settled" : "recap.notSettled")}</Text>
            ) : null}
          </>
        );
      case "day":
        return (
          <>
            <Text variant="caption">{t("recap.favouriteDay")}</Text>
            <Text variant="display" heading>
              {recap.favouriteDay ? day(recap.favouriteDay.date) : ""}
            </Text>
          </>
        );
      case "highlight": {
        const h = recap.highlights[card.index];
        if (!h) return null;
        return (
          <>
            <Text variant="caption">
              {day(h.date)}
              {names[h.authorUid] ? ` · ${names[h.authorUid]}` : ""}
            </Text>
            <Text variant="handwriting">{h.text}</Text>
          </>
        );
      }
      case "closing":
        return (
          <>
            <Text variant="display" heading>
              {t("recapPlayer.thanks")}
            </Text>
            <Button
              label={t("recapPlayer.again")}
              variant="secondary"
              onPress={() => setIndex(0)}
            />
            <Button label={t("recapPlayer.done")} onPress={() => router.back()} />
          </>
        );
    }
  })();

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <View style={{ padding: 20, gap: 12, flex: 1 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <View
            accessible
            accessibilityLabel={t("recapPlayer.position", { n: index + 1, total: cards.length })}
            style={{ flex: 1, flexDirection: "row", gap: 4 }}
          >
            {cards.map((_, i) => (
              <View
                key={i}
                style={{
                  flex: 1,
                  height: 4,
                  borderRadius: 2,
                  backgroundColor: i <= index ? colors.indigo : colors.line,
                }}
              />
            ))}
          </View>
          {close}
        </View>

        <View
          style={{
            flex: 1,
            borderRadius: radius.card,
            backgroundColor: colors.white,
            borderWidth: 1,
            borderColor: colors.line,
            overflow: "hidden",
          }}
        >
          <View style={{ flex: 1, padding: 24, gap: 12, justifyContent: "center" }}>{body}</View>
          {/* Tap zones sit over the card but stay out of the closing card's buttons. */}
          {card.kind === "closing" ? null : (
            <View
              style={{
                position: "absolute",
                top: 0,
                bottom: 0,
                left: 0,
                right: 0,
                flexDirection: "row",
              }}
              accessibilityElementsHidden
              importantForAccessibility="no-hide-descendants"
            >
              <Pressable
                accessibilityRole="none"
                style={{ flex: 1 }}
                onPress={() => go(-1)}
                onPressIn={() => setHeld(true)}
                onPressOut={() => setHeld(false)}
              />
              <Pressable
                accessibilityRole="none"
                style={{ flex: 1 }}
                onPress={() => go(1)}
                onPressIn={() => setHeld(true)}
                onPressOut={() => setHeld(false)}
              />
            </View>
          )}
        </View>

        <View
          style={{ flexDirection: "row", justifyContent: "space-between", alignItems: "center" }}
        >
          <IconButton
            accessibilityLabel={t("recapPlayer.previous")}
            variant="secondary"
            disabled={index === 0}
            onPress={() => go(-1)}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          {reduceMotion ? null : (
            <Button
              label={t(paused ? "recapPlayer.resume" : "recapPlayer.pause")}
              variant="secondary"
              onPress={() => setPaused((p) => !p)}
            />
          )}
          <IconButton
            accessibilityLabel={t("recapPlayer.next")}
            variant="secondary"
            disabled={index === last}
            onPress={() => go(1)}
          >
            <Icon name="forward" color={colors.ink} />
          </IconButton>
        </View>
      </View>
    </SafeAreaView>
  );
}
