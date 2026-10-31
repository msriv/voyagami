import Constants from "expo-constants";
import { getLocales } from "expo-localization";
import { Platform } from "react-native";

/** What support needs to reproduce a problem. Never includes phone numbers, places or free text. */
export function diagnostics(): string {
  return [
    `App ${Constants.expoConfig?.version ?? "unknown"}`,
    `${Platform.OS} ${String(Platform.Version)}`,
    `Language ${getLocales()[0]?.languageTag ?? "unknown"}`,
  ].join("\n");
}
