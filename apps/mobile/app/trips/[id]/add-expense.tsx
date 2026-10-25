import { minorDigits } from "@voyagami/locale";
import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { daysInRange } from "@/trips/dates";
import { parseAmount, todayIn } from "@/trips/money";
import type { Trip } from "@/trips/summary";
import { Button, Chip, IconButton, Input, Text } from "@/ui";

const categories = ["food", "transport", "stay", "activity", "shopping", "other"] as const;
type Category = (typeof categories)[number];

/**
 * Add an expense in the trip's own currency, paid by you and kept to yourself.
 * shortcut: no payer or split choice and no second currency, until the crew list (invites)
 * and multi-currency (VOY-150) exist.
 */
export default function AddExpenseScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [trip, setTrip] = useState<Trip | undefined>();
  const [description, setDescription] = useState("");
  const [amount, setAmount] = useState("");
  const [category, setCategory] = useState<Category | undefined>();
  const [date, setDate] = useState<string | undefined>();
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api
      .unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } }))
      .then(setTrip)
      .catch(() => setError(t("addExpense.errors.generic")));
  }, [api, id, t]);

  const { start, end } = trip?.dates ?? {};
  const days = start && end ? daysInRange(start, end) : [];
  const chosen = date ?? (trip ? (days[0] ?? todayIn(trip.timeZone)) : undefined);

  async function save() {
    if (!trip || !chosen) return;
    const currency = trip.settlementCurrency;
    const minor = parseAmount(amount, currency);
    if (!description.trim()) return setError(t("addExpense.errors.description"));
    if (minor === undefined) {
      return setError(t("addExpense.errors.amount", { digits: minorDigits(currency) }));
    }
    setBusy(true);
    setError(undefined);
    try {
      const me = await api.unwrap(api.GET("/v1/me"));
      await api.unwrap(
        api.POST("/v1/trips/{tripId}/expenses", {
          params: { path: { tripId: id } },
          body: {
            description: description.trim(),
            ...(category ? { category } : null),
            paidByUid: me.id,
            date: chosen,
            original: { amountMinor: minor, currency },
            split: { method: "equal", parts: [{ uid: me.id }] },
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
            <Text variant="bodyStrong">{"‹"}</Text>
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

        <Text variant="caption">{t("addExpense.splitNote")}</Text>
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
