import { formatDate, formatMoney, minorDigits } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useState } from "react";
import { Pressable, ScrollView, Share, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { isoToDate } from "@/trips/dates";
import { parseAmount } from "@/trips/money";
import type { Trip } from "@/trips/summary";
import { Button, Card, Chip, IconButton, Input, Sheet, Text } from "@/ui";

type Expense = components["schemas"]["Expense"];
type Budget = components["schemas"]["BudgetSummary"];

type State =
  | { kind: "loading" }
  | { kind: "ready"; trip: Trip; expenses: Expense[]; budget: Budget; me: string }
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
  const [setting, setSetting] = useState(false);
  const [target, setTarget] = useState("");
  const [scope, setScope] = useState<"perPerson" | "group">("group");

  const load = useCallback(async () => {
    try {
      const [trip, { expenses }, budget, me] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/trips/{tripId}/expenses", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/trips/{tripId}/budget", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/me")),
      ]);
      setState({ kind: "ready", trip, expenses, budget, me: me.id });
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

  const { trip, expenses, budget, me } = state;
  const currency = trip.settlementCurrency;
  const canEdit = trip.role !== "viewer";
  const total = budget.spentMinor;
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
      setState({
        kind: "ready",
        trip,
        me,
        budget: await api.unwrap(
          api.GET("/v1/trips/{tripId}/budget", { params: { path: { tripId: id } } }),
        ),
        expenses: expenses.filter((x) => x.id !== expense.id),
      });
      setOpen(undefined);
    } catch (e) {
      setError(t(e instanceof NetworkError ? "money.errors.offline" : "money.errors.generic"));
    } finally {
      setBusy(false);
    }
  }

  /**
   * Hands the CSV to the phone's share sheet as text.
   * shortcut: shared as text, not a .csv file, until a file-sharing module is added.
   */
  async function exportCsv() {
    setBusy(true);
    setError(undefined);
    try {
      const { data, response } = await api.GET("/v1/trips/{tripId}/expenses/export", {
        params: { path: { tripId: id } },
        parseAs: "text",
      });
      if (!response.ok || typeof data !== "string") throw new ApiError(response.status, undefined);
      await Share.share({ title: t("money.export.title", { name: trip.title }), message: data });
    } catch (e) {
      setError(t(e instanceof NetworkError ? "money.errors.offline" : "money.errors.generic"));
    } finally {
      setBusy(false);
    }
  }

  async function saveBudget() {
    const minor = parseAmount(target, currency);
    if (minor === undefined) {
      return setError(t("money.budget.errors.amount", { digits: minorDigits(currency) }));
    }
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.PATCH("/v1/trips/{tripId}", {
          params: { path: { tripId: id } },
          body: { budget: { scope, amount: { amountMinor: minor, currency } } },
        }),
      );
      setSetting(false);
      await load();
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
          {budget.targetMinor !== undefined && budget.remainingMinor !== undefined ? (
            <Text
              color={budget.overBudget ? "vermilion" : "muted"}
              accessibilityLiveRegion="polite"
            >
              {budget.overBudget
                ? t("money.budget.over", {
                    amount: formatMoney(-budget.remainingMinor, currency, locale),
                    target: formatMoney(budget.targetMinor, currency, locale),
                  })
                : t("money.budget.left", {
                    amount: formatMoney(budget.remainingMinor, currency, locale),
                    target: formatMoney(budget.targetMinor, currency, locale),
                  })}
            </Text>
          ) : null}
          {trip.role === "owner" ? (
            <Button
              label={t(
                budget.targetMinor === undefined ? "money.budget.set" : "money.budget.change",
              )}
              variant="secondary"
              onPress={() => {
                setError(undefined);
                setSetting(true);
              }}
            />
          ) : null}
        </Card>

        {budget.byCategory.length > 0 ? (
          <Card>
            <Text variant="bodyStrong">{t("money.budget.byCategory")}</Text>
            {budget.byCategory.map((c) => (
              <View
                key={c.category}
                accessible
                accessibilityLabel={`${c.category ? t(`addExpense.categories.${c.category}`, { defaultValue: c.category }) : t("money.budget.noCategory")}, ${formatMoney(c.spentMinor, currency, locale)}`}
                style={{ flexDirection: "row", justifyContent: "space-between", gap: 12 }}
              >
                <Text>
                  {c.category
                    ? t(`addExpense.categories.${c.category}`, { defaultValue: c.category })
                    : t("money.budget.noCategory")}
                </Text>
                <Text>{formatMoney(c.spentMinor, currency, locale)}</Text>
              </View>
            ))}
          </Card>
        ) : null}

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

        {expenses.length > 0 ? (
          <>
            <Button
              label={t("money.export.button")}
              variant="secondary"
              loading={busy}
              onPress={() => void exportCsv()}
            />
            {error && !open && !setting ? (
              <Text color="vermilion" accessibilityLiveRegion="polite">
                {error}
              </Text>
            ) : null}
          </>
        ) : null}
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

      <Sheet
        visible={setting}
        onClose={() => setSetting(false)}
        title={t("money.budget.title")}
        closeLabel={t("money.close")}
        closeIcon={<Text>{"×"}</Text>}
      >
        <View style={{ gap: 12, paddingBottom: 12 }}>
          <Input
            label={t("money.budget.amountLabel", { currency })}
            value={target}
            onChangeText={setTarget}
            keyboardType="decimal-pad"
            maxLength={14}
          />
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
            {(["group", "perPerson"] as const).map((k) => (
              <Chip
                key={k}
                label={t(`money.budget.scopes.${k}`)}
                selected={k === scope}
                onPress={() => setScope(k)}
              />
            ))}
          </View>
          <Button label={t("money.budget.save")} loading={busy} onPress={() => void saveBudget()} />
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
