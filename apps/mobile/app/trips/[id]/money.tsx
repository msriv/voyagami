import { formatDate, formatMoney } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { isoToDate } from "@/trips/dates";
import type { Trip } from "@/trips/summary";
import { Button, Card, IconButton, Sheet, Text } from "@/ui";

type Expense = components["schemas"]["Expense"];

type State =
  | { kind: "loading" }
  | { kind: "ready"; trip: Trip; expenses: Expense[]; me: string }
  | { kind: "notFound" }
  | { kind: "error" };

/** Spending on a trip, newest day first. Adding comes from its own screen. */
export default function MoneyScreen() {
  const { t } = useTranslation();
  const { colors, radius, hitArea } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [state, setState] = useState<State>({ kind: "loading" });
  const [open, setOpen] = useState<Expense | undefined>();
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const [trip, { expenses }, me] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/trips/{tripId}/expenses", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/me")),
      ]);
      setState({ kind: "ready", trip, expenses, me: me.id });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  // Also reloads on coming back from adding an expense.
  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  const back = (
    <IconButton
      accessibilityLabel={t("money.back")}
      variant="secondary"
      onPress={() => router.back()}
    >
      <Text variant="bodyStrong">{"‹"}</Text>
    </IconButton>
  );

  if (state.kind !== "ready") {
    return (
      <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi, padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row" }}>{back}</View>
        {state.kind === "loading" ? null : (
          <View style={{ gap: 12 }}>
            <Text>{state.kind === "notFound" ? t("money.notFound") : t("money.loadError")}</Text>
            {state.kind === "error" ? (
              <Button label={t("money.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { trip, expenses, me } = state;
  const currency = trip.settlementCurrency;
  const canEdit = trip.role !== "viewer";
  const total = expenses.reduce((sum, x) => sum + x.settlement.amountMinor, 0);
  const days = [...new Set(expenses.map((x) => x.date))];
  const day = (iso: string) => formatDate(isoToDate(iso), locale, { timeZone: "UTC" });

  async function remove(expense: Expense) {
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/expenses/{expenseId}", {
          params: { path: { tripId: id, expenseId: expense.id } },
        }),
      );
      setState({ kind: "ready", trip, me, expenses: expenses.filter((x) => x.id !== expense.id) });
      setOpen(undefined);
    } catch (e) {
      setError(t(e instanceof NetworkError ? "money.errors.offline" : "money.errors.generic"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          {back}
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("money.title")}
          </Text>
        </View>
        <Text color="muted">{trip.title}</Text>

        <Card>
          <Text variant="caption">{t("money.total")}</Text>
          <Text variant="title" heading>
            {formatMoney(total, currency, locale)}
          </Text>
        </Card>

        {expenses.length === 0 ? (
          <Text color="muted">{t(canEdit ? "money.empty" : "money.emptyViewer")}</Text>
        ) : null}
        {days.map((iso) => (
          <View key={iso} style={{ gap: 8 }}>
            <Text variant="bodyStrong">{day(iso)}</Text>
            {expenses
              .filter((x) => x.date === iso)
              .map((x) => {
                const amount = formatMoney(x.settlement.amountMinor, currency, locale);
                return (
                  <Card key={x.id}>
                    <Pressable
                      accessibilityRole="button"
                      accessibilityLabel={t("money.expenseLabel", {
                        name: x.description,
                        amount,
                        date: day(x.date),
                      })}
                      disabled={!canEdit}
                      onPress={() => {
                        setError(undefined);
                        setOpen(x);
                      }}
                      style={{
                        minHeight: hitArea.min,
                        flexDirection: "row",
                        justifyContent: "space-between",
                        gap: 12,
                        borderRadius: radius.button,
                      }}
                    >
                      <View style={{ flex: 1 }}>
                        <Text variant="bodyStrong">{x.description}</Text>
                        <Text variant="caption">
                          {t(x.paidByUid === me ? "money.paidByYou" : "money.paidByOther")}
                        </Text>
                      </View>
                      <Text variant="bodyStrong">{amount}</Text>
                    </Pressable>
                  </Card>
                );
              })}
          </View>
        ))}

        {canEdit ? (
          <Button label={t("money.add")} onPress={() => router.push(`/trips/${id}/add-expense`)} />
        ) : null}
      </ScrollView>

      <Sheet
        visible={open !== undefined}
        onClose={() => setOpen(undefined)}
        title={open?.description ?? t("money.deleteTitle")}
        closeLabel={t("money.close")}
        closeIcon={<Text>{"×"}</Text>}
      >
        <View style={{ gap: 12, paddingBottom: 12 }}>
          {open ? <Text>{formatMoney(open.settlement.amountMinor, currency, locale)}</Text> : null}
          <Button
            label={t("money.remove")}
            variant="secondary"
            loading={busy}
            onPress={() => open && void remove(open)}
          />
          {error ? (
            <Text color="vermilion" accessibilityLiveRegion="polite">
              {error}
            </Text>
          ) : null}
        </View>
      </Sheet>
    </SafeAreaView>
  );
}
