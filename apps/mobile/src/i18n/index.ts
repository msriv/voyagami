import { getLocales } from "expo-localization";
import i18n from "i18next";
import { initReactI18next, useTranslation } from "react-i18next";

import { en } from "./en";
import { hi } from "./hi";
import { enXA } from "./pseudo";

export const defaultNamespace = "translation";

declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: typeof defaultNamespace;
    resources: { translation: typeof en };
  }
}

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    hi: { translation: hi },
    "en-XA": { translation: enXA },
  },
  lng:
    process.env.EXPO_PUBLIC_PSEUDO_LOCALE === "1"
      ? "en-XA"
      : (getLocales()[0]?.languageCode ?? "en"),
  fallbackLng: "en",
  interpolation: { escapeValue: false },
  returnNull: false,
});

export { i18n, useTranslation };
