import { formatDate, formatMonth, weekdayLabels, firstWeekday } from "@voyagami/locale";
import { useState } from "react";
import { Pressable, View } from "react-native";

import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { IconButton, Text } from "@/ui";

import { daysBetween, isoToDate, monthCells, pickDay, type DateRange } from "./dates";

export interface DateRangePickerProps {
  value: DateRange;
  onChange: (range: DateRange) => void;
  locale: string;
  /** Month shown first, as an ISO day. Defaults to the start of the range, else today. */
  initialMonth?: string;
}

/** A month calendar for choosing a start and an end day. Every day is a button with its full date as label. */
export function DateRangePicker({ value, onChange, locale, initialMonth }: DateRangePickerProps) {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const anchor = isoToDate(initialMonth ?? value.start ?? new Date().toISOString().slice(0, 10));
  const [cursor, setCursor] = useState({ y: anchor.getUTCFullYear(), m: anchor.getUTCMonth() });
  const shift = (by: number) =>
    setCursor(({ y, m }) => {
      const d = new Date(Date.UTC(y, m + by, 1));
      return { y: d.getUTCFullYear(), m: d.getUTCMonth() };
    });
  const cells = monthCells(cursor.y, cursor.m, firstWeekday(locale));
  const summary = value.start
    ? value.end
      ? t("newTrip.rangeSummary", {
          start: formatDate(isoToDate(value.start), locale, { timeZone: "UTC" }),
          end: formatDate(isoToDate(value.end), locale, { timeZone: "UTC" }),
          count: daysBetween(value.start, value.end),
        })
      : t("newTrip.pickEnd")
    : t("newTrip.pickStart");

  return (
    <View
      style={{ backgroundColor: colors.white, borderRadius: 22, padding: 14, gap: 8 }}
      accessibilityLabel={t("newTrip.calendar")}
    >
      <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}>
        <IconButton accessibilityLabel={t("newTrip.prevMonth")} onPress={() => shift(-1)}>
          <Text variant="title">{"‹"}</Text>
        </IconButton>
        <Text variant="bodyStrong" heading>
          {formatMonth(Date.UTC(cursor.y, cursor.m, 1), locale)}
        </Text>
        <IconButton accessibilityLabel={t("newTrip.nextMonth")} onPress={() => shift(1)}>
          <Text variant="title">{"›"}</Text>
        </IconButton>
      </View>
      <Text variant="caption" accessibilityLiveRegion="polite">
        {summary}
      </Text>
      <View style={{ flexDirection: "row" }} importantForAccessibility="no-hide-descendants">
        {weekdayLabels(locale).map((w, i) => (
          <Text key={i} variant="caption" style={{ flex: 1, textAlign: "center" }}>
            {w}
          </Text>
        ))}
      </View>
      <View style={{ flexDirection: "row", flexWrap: "wrap" }}>
        {cells.map((iso, i) => {
          if (!iso) return <View key={`b${i}`} style={{ width: `${100 / 7}%` }} />;
          const isStart = iso === value.start;
          const isEnd = iso === value.end;
          const inside = value.start && value.end && iso > value.start && iso < value.end;
          const selected = isStart || isEnd;
          return (
            <View key={iso} style={{ width: `${100 / 7}%`, alignItems: "center" }}>
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={formatDate(isoToDate(iso), locale, {
                  timeZone: "UTC",
                  withYear: true,
                })}
                accessibilityState={{ selected: Boolean(selected) }}
                hitSlop={2}
                onPress={() => onChange(pickDay(value, iso))}
                style={{
                  width: "100%",
                  minHeight: 44,
                  alignItems: "center",
                  justifyContent: "center",
                  backgroundColor: selected
                    ? colors.vermilion
                    : inside
                      ? colors.line
                      : "transparent",
                  borderRadius: selected ? 12 : 0,
                }}
              >
                <Text variant={selected ? "bodyStrong" : "body"}>{Number(iso.slice(8))}</Text>
              </Pressable>
            </View>
          );
        })}
      </View>
    </View>
  );
}
