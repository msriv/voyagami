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

/** The friendly message plus what actually went wrong, so a tester can say what they saw. */
export function withDetail(message: string, error: unknown, always = false): string {
  // Errors we already explain in plain words need no extra detail.
  if (!always && signInErrorKey(error) !== "generic") return message;
  const detail =
    error instanceof ApiError
      ? `${error.status} ${error.code}: ${error.message}`
      : error instanceof Error
        ? `${error.name}: ${error.message}`
        : String(error);
  return `${message} (${detail})`;
}
