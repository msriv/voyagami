import { minorDigits } from "@voyagami/locale";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { daysInRange } from "@/trips/dates";
import { parseAmount, todayIn } from "@/trips/money";
import type { Trip } from "@/trips/summary";
import { Button, Chip, IconButton, Input, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Member = components["schemas"]["Member"];

const categories = ["food", "transport", "stay", "activity", "shopping", "other"] as const;
type Category = (typeof categories)[number];

/**
 * Add an expense in the trip's own currency: who paid, who shares it (split equally).
 * shortcut: equal splits only and no second currency, until exact, percent and shares splits
 * and multi-currency (VOY-150) are built.
 */
export default function AddExpenseScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [trip, setTrip] = useState<Trip | undefined>();
  const [crew, setCrew] = useState<Member[]>([]);
  const [me, setMe] = useState<string | undefined>();
  const [payer, setPayer] = useState<string | undefined>();
  const [sharers, setSharers] = useState<string[] | undefined>();
  const [description, setDescription] = useState("");
  const [amount, setAmount] = useState("");
  const [category, setCategory] = useState<Category | undefined>();
  const [date, setDate] = useState<string | undefined>();
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

  async function save() {
    if (!trip || !chosen || !paidBy) return;
    const currency = trip.settlementCurrency;
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
            description: description.trim(),
            ...(category ? { category } : null),
            paidByUid: paidBy,
            date: chosen,
            original: { amountMinor: minor, currency },
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
          label={t("addExpense.amountLabel", { currency: trip?.settlementCurrency ?? "" })}
          value={amount}
          onChangeText={setAmount}
          keyboardType="decimal-pad"
          maxLength={14}
        />

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
