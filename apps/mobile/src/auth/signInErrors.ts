import { ApiError, NetworkError } from "@voyagami/api-client";

import type { en } from "@/i18n/en";

export type SignInErrorKey = keyof typeof en.signIn.errors;

/** Maps a failed call to the message to show. Never shows server text: it is not translated. */
export function signInErrorKey(error: unknown): SignInErrorKey {
  if (error instanceof NetworkError) return "offline";
  if (error instanceof ApiError) {
    if (error.code === "invalid_code") return "invalidCode";
    if (error.code === "rate_limited") return "tooMany";
    if (error.code === "bad_request") return "invalidNumber";
  }
  return "generic";
}
