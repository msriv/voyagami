import { defaultCurrency, defaultUnits, localeFor, type Units } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { useAuth } from "@/auth";

interface ProfileValue {
  displayName: string | undefined;
  /** ISO 3166-1 region, or null until the person or the phone gives one. */
  region: string | null;
  /** ISO 4217 code, or "" when unknown. */
  currency: string;
  units: Units;
  /** Locale for formatting dates, numbers and money: device language, home region. */
  locale: string;
  /** Re-reads the profile after a change, so formatting follows at once. */
  reload: () => Promise<void>;
}

const ProfileContext = createContext<ProfileValue | null>(null);

type Me = {
  displayName?: string | undefined;
  homeRegion?: string | undefined;
  homeCurrency?: string | undefined;
  units?: Units & { firstWeekday: number };
  regionSource?: "device" | "confirmed";
};

/**
 * The signed-in person's profile with phone-based fallbacks, so screens always have a region,
 * currency and units to format with, even offline or before the profile loads.
 */
export function ProfileProvider({ children }: { children: ReactNode }) {
  const { api, status } = useAuth();
  const [me, setMe] = useState<Me | undefined>();

  const reload = useCallback(async () => {
    try {
      setMe(await api.unwrap(api.GET("/v1/me")));
    } catch {
      // Keep what we have; screens fall back to the phone's settings.
    }
  }, [api]);

  useEffect(() => {
    if (status === "signedIn") void reload();
    else setMe(undefined);
  }, [status, reload]);

  const value = useMemo<ProfileValue>(() => {
    const device = getLocales()[0];
    const tag = device?.languageTag ?? "en";
    // Until the person confirms, the saved values are placeholders (unknown region and currency),
    // so the phone's settings are used instead.
    const saved = me?.regionSource === "confirmed" ? me : undefined;
    const region = saved?.homeRegion ?? device?.regionCode ?? null;
    return {
      displayName: me?.displayName,
      region,
      currency: saved?.homeCurrency ?? defaultCurrency(region) ?? "",
      units: saved?.units ?? defaultUnits(region, tag),
      locale: localeFor(tag, region),
      reload,
    };
  }, [me, reload]);

  return <ProfileContext.Provider value={value}>{children}</ProfileContext.Provider>;
}

export function useProfile(): ProfileValue {
  const value = useContext(ProfileContext);
  if (!value) throw new Error("useProfile must be used inside ProfileProvider");
  return value;
}
