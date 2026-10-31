import { useRouter } from "expo-router";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { useTranslation } from "@/i18n";
import { onLimitReached } from "@/plans/limitEvents";
import { usePlanPrices } from "@/plans/prices";
import { useTheme } from "@/theme";
import { Button, Sheet, Text } from "@/ui";
import { Icon } from "@/ui/icons";

interface PaywallApi {
  /** Opens the paywall for the plan feature that ran out, like `bestOrder`. */
  show: (feature: string) => void;
}

const Paywall = createContext<PaywallApi | undefined>(undefined);

export function usePaywall(): PaywallApi {
  const api = useContext(Paywall);
  if (!api) throw new Error("usePaywall must be used inside PaywallProvider");
  return api;
}

/** Features that have their own words. Everything else gets the general ones. */
const MOMENTS = ["bestOrder", "multiCurrency"] as const;
type Moment = (typeof MOMENTS)[number];
const momentFor = (feature: string): Moment | "default" =>
  (MOMENTS as readonly string[]).includes(feature) ? (feature as Moment) : "default";

/**
 * The one paywall. Screens never build their own: they call `show(feature)`. It always has a
 * clear "Not now", never blocks the screen underneath, and shows prices only as the store gave them.
 */
export function PaywallProvider({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  const prices = usePlanPrices();
  const [feature, setFeature] = useState<string | undefined>();

  const show = useCallback((f: string) => setFeature(f), []);
  useEffect(() => onLimitReached(show), [show]);
  const api = useMemo(() => ({ show }), [show]);
  const close = () => setFeature(undefined);
  const moment = feature ? momentFor(feature) : "default";

  return (
    <Paywall.Provider value={api}>
      {children}
      <Sheet
        visible={feature !== undefined}
        onClose={close}
        title={t(`paywall.moments.${moment}.title`)}
        closeLabel={t("paywall.notNow")}
        closeIcon={<Icon name="close" color={colors.ink} />}
      >
        <Text>{t(`paywall.moments.${moment}.body`)}</Text>
        <Text variant="caption">{t("paywall.crewNote")}</Text>
        <Button
          label={t("paywall.tryPlus")}
          onPress={() => {
            close();
            router.push("/plans");
          }}
        />
        {prices.pass?.week ? (
          <Text variant="caption">{t("paywall.passFrom", { price: prices.pass.week })}</Text>
        ) : null}
        <Button label={t("paywall.notNow")} variant="ghost" onPress={close} />
        <Text variant="caption">{t("paywall.stillFree")}</Text>
      </Sheet>
    </Paywall.Provider>
  );
}
