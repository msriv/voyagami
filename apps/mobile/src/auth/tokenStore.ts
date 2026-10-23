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
