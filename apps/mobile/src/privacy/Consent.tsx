import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { loadConsents, saveConsents } from "@/auth/tokenStore";

export const CONSENTS = ["location", "notifications", "photos", "analytics"] as const;
export type ConsentKind = (typeof CONSENTS)[number];
type Choices = Record<ConsentKind, boolean>;

/**
 * Location, notifications and photos are still asked by the phone at the moment they are used, so
 * they start on. Analytics is off until the person turns it on.
 */
const DEFAULTS: Choices = { location: true, notifications: true, photos: true, analytics: false };

interface ConsentApi {
  choices: Choices;
  set: (kind: ConsentKind, on: boolean) => void;
}

const Consent = createContext<ConsentApi | undefined>(undefined);

export function useConsent(): ConsentApi {
  const api = useContext(Consent);
  if (!api) throw new Error("useConsent must be used inside ConsentProvider");
  return api;
}

/** Keeps the person's privacy choices on this phone, so they can be changed any time. */
export function ConsentProvider({ children }: { children: ReactNode }) {
  const [choices, setChoices] = useState<Choices>(DEFAULTS);

  useEffect(() => {
    let alive = true;
    void loadConsents()
      .then((saved) => {
        if (alive && saved) setChoices({ ...DEFAULTS, ...(JSON.parse(saved) as Partial<Choices>) });
      })
      .catch(() => undefined);
    return () => {
      alive = false;
    };
  }, []);

  const set = useCallback((kind: ConsentKind, on: boolean) => {
    setChoices((c) => {
      const next = { ...c, [kind]: on };
      void saveConsents(JSON.stringify(next)).catch(() => undefined);
      return next;
    });
  }, []);

  const api = useMemo(() => ({ choices, set }), [choices, set]);
  return <Consent.Provider value={api}>{children}</Consent.Provider>;
}
