import { defaultCurrency, formatMoney, minorDigits } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useRef, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { newId } from "@/local/offlineHook";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { daysInRange } from "@/trips/dates";
import {
  convertMinor,
  parseAmount,
  rateFromMajor,
  rateToMajor,
  todayIn,
  type RateRatio,
} from "@/trips/money";
import type { Trip } from "@/trips/summary";
import { Button, Chip, IconButton, Input, Text } from "@/ui";
import { Icon } from "@/ui/icons";

/** True when the platform knows the ISO 4217 code. */
function validCurrency(code: string): boolean {
  try {
    minorDigits(code);
    return true;
  } catch {
    return false;
  }
}

type Member = components["schemas"]["Member"];

const categories = ["food", "transport", "stay", "activity", "shopping", "other"] as const;
type Category = (typeof categories)[number];

/**
 * Add an expense in the trip's currency or another one (a suggested rate converts it, and the person
 * can type their own): who paid, who shares it (split equally).
 * shortcut: equal splits only, until exact, percent and shares splits are built.
 */
export default function AddExpenseScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  // One ID per form, so a retry or a queued resend can never add the expense twice.
  const expenseId = useRef(newId());
  const [trip, setTrip] = useState<Trip | undefined>();
  const [crew, setCrew] = useState<Member[]>([]);
  const [me, setMe] = useState<string | undefined>();
  const [payer, setPayer] = useState<string | undefined>();
  const [sharers, setSharers] = useState<string[] | undefined>();
  const [description, setDescription] = useState("");
  const [amount, setAmount] = useState("");
  const [category, setCategory] = useState<Category | undefined>();
  const [date, setDate] = useState<string | undefined>();
  const [currencyPick, setCurrencyPick] = useState<string | undefined>();
  const [otherCode, setOtherCode] = useState("");
  const [suggested, setSuggested] = useState<RateRatio | "none" | undefined>();
  const [typedRate, setTypedRate] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    Promise.all([
      api.unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } })),
      api.unwrap(api.GET("/v1/trips/{tripId}/members", { params: { path: { tripId: id } } })),
      api.unwrap(api.GET("/v1/me")),
    ])
      .then(([tripBody, { members }, meBody]) => {
        setTrip(tripBody);
        setCrew(members.filter((m) => m.status === "active"));
        setMe(meBody.id);
      })
      .catch(() => setError(t("addExpense.errors.generic")));
  }, [api, id, t]);

  const paidBy = payer ?? me;
  const sharedBy = sharers ?? crew.map((m) => m.uid);
  const who = (m: Member) => (m.uid === me ? t("addExpense.you") : m.displayName);
  const toggleSharer = (uid: string) =>
    setSharers(sharedBy.includes(uid) ? sharedBy.filter((u) => u !== uid) : [...sharedBy, uid]);

  const { start, end } = trip?.dates ?? {};
  const days = start && end ? daysInRange(start, end) : [];
  const chosen = date ?? (trip ? (days[0] ?? todayIn(trip.timeZone)) : undefined);

  const settlement = trip?.settlementCurrency;
  const options = [
    ...new Set([
      ...(settlement ? [settlement] : []),
      ...(trip?.destinations ?? []).flatMap((d) => defaultCurrency(d.countryCode) ?? []),
    ]),
  ];
  const typedCode = otherCode.trim().toUpperCase();
  const currency =
    currencyPick === "other"
      ? /^[A-Z]{3}$/.test(typedCode) && validCurrency(typedCode)
        ? typedCode
        : undefined
      : (currencyPick ?? settlement);
  const foreign = !!currency && !!settlement && currency !== settlement;

  useEffect(() => {
    setSuggested(undefined);
    setTypedRate("");
    if (!foreign || !currency || !settlement) return;
    let live = true;
    api
      .unwrap(api.GET("/v1/rates", { params: { query: { from: currency, to: settlement } } }))
      .then((q) => live && setSuggested(q.rate))
      .catch(() => live && setSuggested("none"));
    return () => {
      live = false;
    };
  }, [api, foreign, currency, settlement]);

  const typed =
    foreign && currency && settlement ? rateFromMajor(typedRate, currency, settlement) : undefined;
  const rate: RateRatio | undefined =
    typed ?? (suggested && suggested !== "none" ? suggested : undefined);
  const minorNow = currency ? parseAmount(amount, currency) : undefined;
  const locale = getLocales()[0]?.languageTag ?? "en";

  async function save() {
    if (!trip || !chosen || !paidBy || !settlement) return;
    if (!currency) return setError(t("addExpense.errors.currency"));
    if (foreign && !rate) return setError(t("addExpense.errors.rate"));
    const minor = parseAmount(amount, currency);
    if (!description.trim()) return setError(t("addExpense.errors.description"));
    if (minor === undefined) {
      return setError(t("addExpense.errors.amount", { digits: minorDigits(currency) }));
    }
    if (sharedBy.length === 0) return setError(t("addExpense.errors.sharers"));
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.POST("/v1/trips/{tripId}/expenses", {
          params: { path: { tripId: id } },
          body: {
            id: expenseId.current,
            description: description.trim(),
            ...(category ? { category } : null),
            paidByUid: paidBy,
            date: chosen,
            original: { amountMinor: minor, currency },
            ...(foreign && rate ? { rate } : null),
            split: { method: "equal", parts: sharedBy.map((uid) => ({ uid })) },
          },
        }),
      );
      router.back();
    } catch (e) {
      setError(
        t(e instanceof NetworkError ? "addExpense.errors.offline" : "addExpense.errors.generic"),
      );
    } finally {
      setBusy(false);
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
            accessibilityLabel={t("addExpense.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("addExpense.title")}
          </Text>
        </View>

        <Input
          label={t("addExpense.descriptionLabel")}
          value={description}
          onChangeText={setDescription}
          maxLength={120}
        />
        <Input
          label={t("addExpense.amountLabel", { currency: currency ?? "" })}
          value={amount}
          onChangeText={setAmount}
          keyboardType="decimal-pad"
          maxLength={14}
        />

        <View style={{ gap: 6 }}>
          <Text variant="bodyStrong">{t("addExpense.currencyLabel")}</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
            {options.map((c) => (
              <Chip
                key={c}
                label={c}
                selected={c === currency && currencyPick !== "other"}
                onPress={() => setCurrencyPick(c)}
              />
            ))}
            <Chip
              label={t("addExpense.otherCurrency")}
              selected={currencyPick === "other"}
              onPress={() => setCurrencyPick("other")}
            />
          </View>
          {currencyPick === "other" ? (
            <Input
              label={t("addExpense.otherCurrencyLabel")}
              value={otherCode}
              onChangeText={setOtherCode}
              autoCapitalize="characters"
              autoCorrect={false}
              maxLength={3}
            />
          ) : null}
        </View>

        {foreign && currency && settlement ? (
          <View style={{ gap: 6 }}>
            {rate ? (
              <Text accessibilityLiveRegion="polite">
                {t(typed ? "addExpense.rateYours" : "addExpense.rateSuggested", {
                  from: formatMoney(10 ** minorDigits(currency), currency, locale),
                  to: formatMoney(
                    convertMinor(10 ** minorDigits(currency), rate),
                    settlement,
                    locale,
                  ),
                })}
              </Text>
            ) : suggested === "none" ? (
              <Text accessibilityLiveRegion="polite">
                {t("addExpense.rateMissing", { from: currency, to: settlement })}
              </Text>
            ) : (
              <Text color="muted">{t("addExpense.rateLoading")}</Text>
            )}
            {rate && minorNow ? (
              <Text variant="bodyStrong">
                {t("addExpense.converted", {
                  amount: formatMoney(convertMinor(minorNow, rate), settlement, locale),
                })}
              </Text>
            ) : null}
            <Input
              label={t("addExpense.rateInput", { from: currency, to: settlement })}
              value={typedRate}
              onChangeText={setTypedRate}
              keyboardType="decimal-pad"
              maxLength={14}
              placeholder={
                suggested && suggested !== "none"
                  ? String(rateToMajor(suggested, currency, settlement))
                  : ""
              }
            />
          </View>
        ) : null}

        <View style={{ gap: 6 }}>
          <Text variant="bodyStrong">{t("addExpense.categoryLabel")}</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
            {categories.map((c) => (
              <Chip
                key={c}
                label={t(`addExpense.categories.${c}`)}
                selected={c === category}
                onPress={() => setCategory(c === category ? undefined : c)}
              />
            ))}
          </View>
        </View>

        {days.length > 1 ? (
          <View style={{ gap: 6 }}>
            <Text variant="bodyStrong">{t("addExpense.dayLabel")}</Text>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={{ gap: 8 }}
            >
              {days.map((d, i) => (
                <Chip
                  key={d}
                  label={t("addExpense.dayChip", { n: i + 1 })}
                  selected={d === chosen}
                  onPress={() => setDate(d)}
                />
              ))}
            </ScrollView>
          </View>
        ) : null}

        {crew.length > 1 ? (
          <>
            <View style={{ gap: 6 }}>
              <Text variant="bodyStrong">{t("addExpense.paidByLabel")}</Text>
              <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
                {crew.map((m) => (
                  <Chip
                    key={m.uid}
                    label={who(m)}
                    selected={m.uid === paidBy}
                    onPress={() => setPayer(m.uid)}
                  />
                ))}
              </View>
            </View>
            <View style={{ gap: 6 }}>
              <Text variant="bodyStrong">{t("addExpense.sharedByLabel")}</Text>
              <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
                {crew.map((m) => (
                  <Chip
                    key={m.uid}
                    label={who(m)}
                    selected={sharedBy.includes(m.uid)}
                    onPress={() => toggleSharer(m.uid)}
                  />
                ))}
              </View>
              <Text variant="caption">{t("addExpense.splitNote")}</Text>
            </View>
          </>
        ) : null}
        {error ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}
        <Button label={t("addExpense.save")} loading={busy} onPress={() => void save()} />
      </ScrollView>
    </SafeAreaView>
  );
}
