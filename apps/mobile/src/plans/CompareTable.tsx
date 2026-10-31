import { useEffect, useState } from "react";
import { useWindowDimensions, View } from "react-native";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { Card, Text } from "@/ui";

/** Order of rows. Counted features first, then on/off ones. */
const COUNTED = [
  "upcomingTrips",
  "bestOrder",
  "forwardedBookings",
  "files",
  "journalPhotos",
  "topPicks",
  "askDrafts",
  "recapSeconds",
] as const;
const FLAGS = [
  "multiCurrency",
  "liveCrewMap",
  "receiptScans",
  "offlineMaps",
  "gmailImport",
  "darkMode",
] as const;

/** Past this text scale the columns stack so nothing clips. */
const STACK_AT = 1.3;

/**
 * Free next to Plus. Free values come from the server's entitlements table,
 * never typed in here. Plus lifts every limit.
 */
export function CompareTable() {
  const { t } = useTranslation();
  const { api } = useAuth();
  const { fontScale } = useWindowDimensions();
  const [free, setFree] = useState<Record<string, { limit: number }> | undefined>();

  useEffect(() => {
    api
      .unwrap(api.GET("/v1/plans/free"))
      .then((e) => setFree(e.features))
      .catch(() => setFree(undefined));
  }, [api]);

  if (!free) return null;
  const stacked = fontScale > STACK_AT;

  const freeValue = (key: string, flag: boolean) => {
    const limit = free[key]?.limit;
    if (limit === undefined) return undefined;
    if (limit === -1) return flag ? t("plans.compare.included") : t("plans.compare.noLimit");
    if (limit === 0) return t("plans.compare.notIncluded");
    return key === "recapSeconds" ? t("plans.compare.seconds", { n: limit }) : String(limit);
  };
  const plusValue = (flag: boolean) =>
    flag ? t("plans.compare.included") : t("plans.compare.noLimit");

  const rows = [
    ...COUNTED.map((k) => ({ k, flag: false })),
    ...FLAGS.map((k) => ({ k, flag: true })),
  ].flatMap(({ k, flag }) => {
    const f = freeValue(k, flag);
    return f === undefined ? [] : [{ k, f, p: plusValue(flag) }];
  });

  return (
    <Card style={{ gap: 4 }}>
      <Text variant="bodyStrong" heading accessibilityRole="header">
        {t("plans.compare.title")}
      </Text>
      {rows.map(({ k, f, p }) => {
        const name = t(`plans.compare.rows.${k}`);
        return (
          <View
            key={k}
            accessible
            accessibilityLabel={t("plans.compare.row", { name, free: f, plus: p })}
            style={{
              flexDirection: stacked ? "column" : "row",
              gap: stacked ? 2 : 12,
              paddingVertical: 8,
            }}
          >
            <Text style={{ flex: stacked ? undefined : 2 }}>{name}</Text>
            <Text variant="caption" style={{ flex: stacked ? undefined : 1 }}>
              {t("plans.compare.cell", { plan: t("plans.compare.free"), value: f })}
            </Text>
            <Text variant="caption" style={{ flex: stacked ? undefined : 1 }}>
              {t("plans.compare.cell", { plan: t("plans.compare.plus"), value: p })}
            </Text>
          </View>
        );
      })}
    </Card>
  );
}
