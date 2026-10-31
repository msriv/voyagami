import { createContext, useContext } from "react";

/**
 * Prices exactly as the store gives them (already in the person's currency and format). Nothing
 * here is ever typed in by us: the purchases setup (RevenueCat, VOY-188) provides them. Until it
 * does, screens say the price is shown by the store.
 */
export interface PlanPrices {
  plusYearly?: string;
  pass?: { weekend?: string; week?: string; long?: string };
  /** Asks the store to give back earlier purchases. Absent when purchases are not set up. */
  restore?: () => Promise<void>;
}

const Prices = createContext<PlanPrices>({});

export const PlanPricesProvider = Prices.Provider;
export const usePlanPrices = () => useContext(Prices);
