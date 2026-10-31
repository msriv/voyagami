import { formatTime } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useState } from "react";
import { View } from "react-native";

import { ApiError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { Button, Text } from "@/ui";

type Status = components["schemas"]["FlightStatus"];
type Leg = components["schemas"]["FlightLeg"];
type Result = { kind: "ok"; status: Status } | { kind: "unavailable" } | { kind: "error" };

/** Live status for one flight booking, fetched when the person asks. Times use the phone's clock setting. */
export function FlightStatus({ tripId, bookingId }: { tripId: string; bookingId: string }) {
  const { t } = useTranslation();
  const { api } = useAuth();
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<Result | undefined>();
  const locale = getLocales()[0]?.languageTag ?? "en";

  async function check() {
    setBusy(true);
    try {
      const status = await api.unwrap(
        api.GET("/v1/trips/{tripId}/bookings/{bookingId}/flight-status", {
          params: { path: { tripId, bookingId } },
        }),
      );
      setResult({ kind: "ok", status });
    } catch (e) {
      // 503 means no status provider is set up yet.
      setResult(
        e instanceof ApiError && e.status === 503 ? { kind: "unavailable" } : { kind: "error" },
      );
    } finally {
      setBusy(false);
    }
  }

  const time = (iso?: string) => (iso ? formatTime(new Date(iso), locale) : undefined);
  const leg = (name: "departure" | "arrival", l?: Leg) => {
    if (!l) return null;
    const planned = time(l.scheduled);
    const now = time(l.estimated);
    const parts = [
      planned ? t(`flight.${name}`, { time: planned }) : undefined,
      now && now !== planned ? t("flight.expected", { time: now }) : undefined,
      l.terminal ? t("flight.terminal", { terminal: l.terminal }) : undefined,
      l.gate ? t("flight.gate", { gate: l.gate }) : undefined,
    ].filter(Boolean);
    return parts.length ? <Text variant="caption">{parts.join(" · ")}</Text> : null;
  };

  return (
    <View style={{ gap: 4, marginTop: 8 }}>
      <Button
        label={t("flight.check")}
        variant="secondary"
        loading={busy}
        onPress={() => void check()}
      />
      <View accessibilityLiveRegion="polite" style={{ gap: 2 }}>
        {result?.kind === "ok" ? (
          <>
            <Text variant="bodyStrong">{t(`flight.states.${result.status.state}`)}</Text>
            {leg("departure", result.status.departure)}
            {leg("arrival", result.status.arrival)}
          </>
        ) : null}
        {result?.kind === "unavailable" ? (
          <Text variant="caption">{t("flight.unavailable")}</Text>
        ) : null}
        {result?.kind === "error" ? <Text variant="caption">{t("flight.error")}</Text> : null}
      </View>
    </View>
  );
}
