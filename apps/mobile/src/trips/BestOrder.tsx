import { useEffect, useState } from "react";
import { View } from "react-native";

import { NetworkError } from "@voyagami/api-client";
import type { components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { Button, Sheet, Text } from "@/ui";

import { applyOrder, stopsOn, type Stop } from "./stops";

type Suggestion = components["schemas"]["BestOrder"];

interface Props {
  tripId: string;
  date: string;
  /** Every stop on the trip; this works on the ones for `date`. */
  stops: Stop[];
  /** Called with the whole list whenever the order changes, including a rollback. */
  onStops: (next: Stop[]) => void;
}

/** "Suggest the best order" for one day: preview, use it, and undo. Only for people who can edit. */
export function BestOrder({ tripId, date, stops, onStops }: Props) {
  const { t } = useTranslation();
  const { api } = useAuth();
  const [suggestion, setSuggestion] = useState<Suggestion | undefined>();
  const [undoTo, setUndoTo] = useState<string[] | undefined>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | undefined>();

  // An undo only makes sense for the day it was made on.
  useEffect(() => setUndoTo(undefined), [date]);

  const day = stopsOn(stops, date);
  if (day.length < 2) return null;

  const fail = (e: unknown) =>
    setError(
      t(
        e instanceof NetworkError
          ? "itinerary.best.errors.offline"
          : "itinerary.best.errors.generic",
      ),
    );

  async function suggest() {
    setBusy(true);
    setError(undefined);
    try {
      setSuggestion(
        await api.unwrap(
          api.POST("/v1/trips/{tripId}/days/{date}/best-order", {
            params: { path: { tripId, date } },
          }),
        ),
      );
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  /** Sets the day's order at once and rolls back if the server says no. Returns whether it stuck. */
  async function setOrder(ids: string[]): Promise<boolean> {
    const before = stops;
    setError(undefined);
    onStops(applyOrder(stops, date, ids));
    try {
      await api.unwrap(
        api.PUT("/v1/trips/{tripId}/days/{date}/order", {
          params: { path: { tripId, date } },
          body: { stopIds: ids },
        }),
      );
      return true;
    } catch (e) {
      onStops(before);
      fail(e);
      return false;
    }
  }

  async function use(s: Suggestion) {
    const old = day.map((x) => x.id);
    setSuggestion(undefined);
    if (await setOrder(s.stopIds)) setUndoTo(old);
  }

  async function undo(ids: string[]) {
    setUndoTo(undefined);
    await setOrder(ids);
  }

  return (
    <View style={{ gap: 8 }}>
      <Button
        label={t("itinerary.best.button")}
        variant="secondary"
        loading={busy}
        onPress={() => void suggest()}
      />
      {undoTo ? (
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <Text style={{ flex: 1 }}>{t("itinerary.best.applied")}</Text>
          <Button
            label={t("itinerary.best.undo")}
            variant="secondary"
            onPress={() => void undo(undoTo)}
          />
        </View>
      ) : null}
      {error ? (
        <Text color="vermilion" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : null}

      <Sheet
        visible={suggestion !== undefined}
        onClose={() => setSuggestion(undefined)}
        title={t("itinerary.best.title")}
        closeLabel={t("itinerary.best.close")}
        closeIcon={<Text>{"×"}</Text>}
      >
        {suggestion ? (
          <View style={{ gap: 12, paddingBottom: 12 }}>
            {suggestion.changed ? (
              <>
                <Text>
                  {t("itinerary.best.summary", {
                    before: suggestion.beforeMin,
                    after: suggestion.afterMin,
                  })}
                </Text>
                {suggestion.stopIds.map((id, i) => (
                  <Text key={id} variant="bodyStrong">
                    {`${i + 1}. ${day.find((s) => s.id === id)?.place.name ?? ""}`}
                  </Text>
                ))}
                <Text variant="caption">{t("itinerary.best.note")}</Text>
                <Button label={t("itinerary.best.apply")} onPress={() => void use(suggestion)} />
                <Button
                  label={t("itinerary.best.keep")}
                  variant="secondary"
                  onPress={() => setSuggestion(undefined)}
                />
              </>
            ) : (
              <>
                <Text>{t("itinerary.best.same")}</Text>
                <Button
                  label={t("itinerary.best.keep")}
                  variant="secondary"
                  onPress={() => setSuggestion(undefined)}
                />
              </>
            )}
          </View>
        ) : null}
      </Sheet>
    </View>
  );
}
