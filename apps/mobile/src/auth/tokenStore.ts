import type { StoredTokens } from "@voyagami/api-client";
import * as SecureStore from "expo-secure-store";

const accessKey = "voyagami.accessToken";
const refreshKey = "voyagami.refreshToken";

/** Tokens live in the Keychain (iOS) or Keystore (Android), never in plain storage. */
export async function loadTokens(): Promise<StoredTokens | null> {
  const [accessToken, refreshToken] = await Promise.all([
    SecureStore.getItemAsync(accessKey),
    SecureStore.getItemAsync(refreshKey),
  ]);
  return accessToken && refreshToken ? { accessToken, refreshToken } : null;
}

export async function saveTokens(tokens: StoredTokens | null): Promise<void> {
  if (tokens === null) {
    await Promise.all([
      SecureStore.deleteItemAsync(accessKey),
      SecureStore.deleteItemAsync(refreshKey),
    ]);
    return;
  }
  await Promise.all([
    SecureStore.setItemAsync(accessKey, tokens.accessToken),
    SecureStore.setItemAsync(refreshKey, tokens.refreshToken),
  ]);
}

const onboardedKey = "voyagami.onboarded";

/** Whether first-run onboarding was finished on this device. */
export async function loadOnboarded(): Promise<boolean> {
  return (await SecureStore.getItemAsync(onboardedKey)) === "1";
}

export async function saveOnboarded(done: boolean): Promise<void> {
  if (done) await SecureStore.setItemAsync(onboardedKey, "1");
  else await SecureStore.deleteItemAsync(onboardedKey);
}

/** Forgets everything kept on this device, so the next launch starts at Welcome. */
export async function clearDevice(): Promise<void> {
  await Promise.all([saveTokens(null), saveOnboarded(false)]);
}
