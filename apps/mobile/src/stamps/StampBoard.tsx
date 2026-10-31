import { useFocusEffect } from "expo-router";
import { useCallback, useRef, useState } from "react";
import { View } from "react-native";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Card, Text } from "@/ui";

type Kind = "createTrip" | "inviteCrew" | "addPlaces" | "logExpense" | "writeJournal";

interface Stamp {
  kind: Kind;
  earned: boolean;
}

/**
 * The first five stamps, one for each part of the app. It lives on Home until all five are earned.
 * A stamp earned since the last look is announced to screen readers; nothing moves, so there is
 * no motion to tone down for reduce motion.
 */
export function StampBoard() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const [stamps, setStamps] = useState<Stamp[] | undefined>();
  const [news, setNews] = useState<Kind | undefined>();
  const seen = useRef<Set<Kind> | undefined>(undefined);

  useFocusEffect(
    useCallback(() => {
      let alive = true;
      api
        .unwrap(api.GET("/v1/me/stamps"))
        .then((r) => {
          if (!alive) return;
          const earned = new Set(r.stamps.filter((s) => s.earned).map((s) => s.kind));
          const fresh = [...earned].filter((k) => seen.current && !seen.current.has(k));
          seen.current = earned;
          setNews(fresh[0]);
          setStamps(r.stamps);
        })
        .catch(() => undefined);
      return () => {
        alive = false;
      };
    }, [api]),
  );

  if (!stamps || stamps.every((s) => s.earned)) return null;
  const count = stamps.filter((s) => s.earned).length;

  return (
    <Card style={{ gap: 10 }}>
      <Text variant="bodyStrong" heading accessibilityRole="header">
        {t("stamps.title")}
      </Text>
      <Text variant="caption">{t("stamps.intro")}</Text>
      {news ? (
        <Text accessibilityLiveRegion="polite" style={{ color: colors.indigo, fontWeight: "800" }}>
          {t("stamps.earnedNow", { name: t(`stamps.kinds.${news}.title`) })}
        </Text>
      ) : null}
      {stamps.map((s) => (
        <View
          key={s.kind}
          accessible
          accessibilityLabel={t(s.earned ? "stamps.rowEarned" : "stamps.rowOpen", {
            name: t(`stamps.kinds.${s.kind}.title`),
            hint: t(`stamps.kinds.${s.kind}.hint`),
          })}
          style={{ flexDirection: "row", alignItems: "center", gap: 12, minHeight: 44 }}
        >
          <View
            style={{
              width: 28,
              height: 28,
              borderRadius: 14,
              borderWidth: 2,
              borderColor: colors.indigo,
              backgroundColor: s.earned ? colors.matcha : "transparent",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            {s.earned ? <Text variant="caption">{t("stamps.tick")}</Text> : null}
          </View>
          <View style={{ flex: 1 }}>
            <Text variant="bodyStrong">{t(`stamps.kinds.${s.kind}.title`)}</Text>
            <Text variant="caption">{t(`stamps.kinds.${s.kind}.hint`)}</Text>
          </View>
        </View>
      ))}
      <Text variant="caption">{t("stamps.count", { earned: count, total: stamps.length })}</Text>
    </Card>
  );
}
