import { formatMoney } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Card, IconButton, Text } from "@/ui";

type Balances = components["schemas"]["Balances"];
type Settlement = components["schemas"]["Settlement"];

type State =
  | { kind: "loading" }
  | {
      kind: "ready";
      balances: Balances;
      settlements: Settlement[];
      names: Record<string, string>;
      me: string;
    }
  | { kind: "notFound" }
  | { kind: "error" };

/** Who owes whom, with the short list of payments that clears everyone. Voyagami never moves money. */
export default function SettleScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const [state, setState] = useState<State>({ kind: "loading" });
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState<string | undefined>();

  const load = useCallback(async () => {
    try {
      const path = { params: { path: { tripId: id } } };
      const [balances, { settlements }, { members }, me] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}/balances", path)),
        api.unwrap(api.GET("/v1/trips/{tripId}/settlements", path)),
        api.unwrap(api.GET("/v1/trips/{tripId}/members", path)),
        api.unwrap(api.GET("/v1/me")),
      ]);
      const names = Object.fromEntries(members.map((m) => [m.uid, m.displayName ?? ""]));
      setState({ kind: "ready", balances, settlements, names, me: me.id });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  const back = (
    <IconButton
      accessibilityLabel={t("settle.back")}
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
            <Text>{state.kind === "notFound" ? t("settle.notFound") : t("settle.loadError")}</Text>
            {state.kind === "error" ? (
              <Button label={t("settle.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { balances, settlements, names, me } = state;
  const { currency } = balances;
  const money = (n: number) => formatMoney(n, currency, locale);
  const who = (uid: string) => (uid === me ? t("settle.you") : names[uid] || t("settle.someone"));

  async function markPaid(fromUid: string, toUid: string, amountMinor: number) {
    const key = `${fromUid}>${toUid}`;
    setBusy(key);
    setError(undefined);
    try {
      await api.unwrap(
        api.POST("/v1/trips/{tripId}/settlements", {
          params: { path: { tripId: id } },
          body: { fromUid, toUid, amountMinor, method: "manual" },
        }),
      );
      await load();
    } catch (e) {
      setError(t(e instanceof NetworkError ? "settle.errors.offline" : "settle.errors.generic"));
    } finally {
      setBusy(undefined);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          {back}
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("settle.title")}
          </Text>
        </View>

        {balances.transfers.length === 0 ? (
          <Card>
            <Text variant="bodyStrong">{t("settle.allSquare")}</Text>
          </Card>
        ) : (
          <View style={{ gap: 12 }}>
            <Text variant="bodyStrong">{t("settle.toPay")}</Text>
            {balances.transfers.map((x) => {
              const key = `${x.fromUid}>${x.toUid}`;
              const mine = x.fromUid === me || x.toUid === me;
              return (
                <Card key={key}>
                  <Text>
                    {t("settle.transfer", {
                      from: who(x.fromUid),
                      to: who(x.toUid),
                      amount: money(x.amountMinor),
                    })}
                  </Text>
                  {mine ? (
                    <Button
                      label={t("settle.markPaid")}
                      variant="secondary"
                      loading={busy === key}
                      onPress={() => void markPaid(x.fromUid, x.toUid, x.amountMinor)}
                    />
                  ) : null}
                </Card>
              );
            })}
          </View>
        )}
        {error ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}

        <Card>
          <Text variant="bodyStrong">{t("settle.balances")}</Text>
          {balances.balances.map((b) => (
            <View
              key={b.uid}
              accessible
              style={{ flexDirection: "row", justifyContent: "space-between", gap: 12 }}
            >
              <Text style={{ flex: 1 }}>{who(b.uid)}</Text>
              <Text color={b.amountMinor < 0 ? "vermilion" : "muted"}>
                {b.amountMinor === 0
                  ? t("settle.even")
                  : t(b.amountMinor > 0 ? "settle.getsBack" : "settle.owes", {
                      amount: money(Math.abs(b.amountMinor)),
                    })}
              </Text>
            </View>
          ))}
        </Card>

        {settlements.length > 0 ? (
          <Card>
            <Text variant="bodyStrong">{t("settle.history")}</Text>
            {settlements.map((s) => (
              <Text key={s.id} color="muted">
                {t("settle.paid", {
                  from: who(s.fromUid),
                  to: who(s.toUid),
                  amount: formatMoney(s.amount.amountMinor, s.amount.currency, locale),
                })}
              </Text>
            ))}
          </Card>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}
