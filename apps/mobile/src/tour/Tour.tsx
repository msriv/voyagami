import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { View } from "react-native";

import { useAuth } from "@/auth";
import { loadTour, saveTour } from "@/auth/tokenStore";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Text } from "@/ui";

/** Keys under `tour.steps` in the strings. One card per step, in order. */
export const TOUR_STEPS = ["trip", "days", "crew"] as const;

interface TourApi {
  /** Starts the tour again from the first step. */
  restart: () => void;
}

const Tour = createContext<TourApi | undefined>(undefined);

export function useTour(): TourApi {
  const api = useContext(Tour);
  if (!api) throw new Error("useTour must be used inside TourProvider");
  return api;
}

/**
 * A short tour of coach cards over the real screens. A card floats above the screen instead of
 * pointing at one control, so it cannot land in the wrong place when the layout changes, and it
 * never blocks what is underneath. It can be skipped at any step and remembers where it stopped.
 */
export function TourProvider({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { status, onboarded } = useAuth();
  const [step, setStep] = useState<number | "done" | undefined>();

  useEffect(() => {
    let alive = true;
    void loadTour()
      .catch(() => "done" as const)
      .then((saved) => alive && setStep(saved ?? 0));
    return () => {
      alive = false;
    };
  }, []);

  const move = useCallback((next: number | "done") => {
    setStep(next);
    void saveTour(next).catch(() => undefined);
  }, []);
  const api = useMemo(() => ({ restart: () => move(0) }), [move]);

  const showing = status === "signedIn" && onboarded && typeof step === "number";
  const key = typeof step === "number" ? TOUR_STEPS[step] : undefined;
  const last = typeof step === "number" && step === TOUR_STEPS.length - 1;

  return (
    <Tour.Provider value={api}>
      {children}
      {showing && key !== undefined && typeof step === "number" ? (
        <View
          pointerEvents="box-none"
          style={{ position: "absolute", left: 16, right: 16, bottom: 96 }}
        >
          <View
            accessibilityLiveRegion="polite"
            style={{
              backgroundColor: colors.indigo,
              borderRadius: 24,
              padding: 16,
              gap: 8,
            }}
          >
            <View style={{ flexDirection: "row", justifyContent: "space-between", gap: 12 }}>
              <Text variant="caption" color="white" style={{ fontWeight: "800" }}>
                {t("tour.progress", { n: step + 1, total: TOUR_STEPS.length })}
              </Text>
              <Button label={t("tour.skip")} variant="ghost" onPress={() => move("done")} />
            </View>
            <Text variant="bodyStrong" color="white" heading>
              {t(`tour.steps.${key}.title`)}
            </Text>
            <Text color="white">{t(`tour.steps.${key}.body`)}</Text>
            <View style={{ flexDirection: "row", gap: 12 }}>
              {step > 0 ? (
                <Button label={t("tour.back")} variant="secondary" onPress={() => move(step - 1)} />
              ) : null}
              <Button
                label={last ? t("tour.finish") : t("tour.next")}
                onPress={() => move(last ? "done" : step + 1)}
              />
            </View>
          </View>
        </View>
      ) : null}
    </Tour.Provider>
  );
}
