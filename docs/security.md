# Security

> Stub. Grows with the D0 auth tickets and is hardened in P1. Report anything that looks wrong to Mihir rather than in a public issue.

## Secrets

- Never commit secrets. `.env` files, keys, service-account JSON and Firebase config files are git-ignored; commit an `.env.example` with placeholders instead.
- CI and Terraform reach Google Cloud through workload identity federation. No long-lived cloud keys.
- Claude's PRs are opened by the `voyagami-bot` GitHub App. Its private key is only a GitHub Actions secret, never in a Claude environment. See [github-bot.md](github-bot.md).
- D phases use only the `voyagami-dev` project, emulators and stubs (SMS code `123456`). No staging, prod or paid third-party accounts until P1.
- From P1 there are two projects: `voyagami-dev` (dev and staging) and `voyagami-prod` (production only). See `infra/README.md`.

## Auth

- Phone OTP behind a provider interface (MSG91 for India, another provider elsewhere; a stub in dev). See `services/api/internal/otp`: `SMSProvider`, the dev `Stub` (only when `APP_ENV=dev`) and the per-number `Limited` wrapper.
- The server decides every limit through the entitlements service. The app only reflects the answer.
- AI calls (Gemini on Vertex) go only through the backend.

## Privacy

- No phone numbers, exact locations or free text in logs or analytics.
- Location sharing is opt-in and ends automatically.
- Photo EXIF location is stripped or kept only as the user chooses.

## Firestore rules

`firestore.rules` denies every direct read and write from a client, on every path, including paths that are not in the data model. All access goes through the Go API, which uses the Admin SDK and is not affected by the rules. `packages/rules-tests` proves this on the Firestore emulator for every path in [data-model.md](data-model.md), for signed-out clients, signed-in strangers and a trip's owner (membership alone opens nothing). Run them with `make test`.

When a later ticket needs direct client access (live co-editing in D2, local-first sync in D3), it opens one collection at a time: a rule decided from `trips/{tripId}/members/{uid}`, plus tests for each role and for a removed member, and an update to the table in the data model.

## Sign-in tokens

- Phone sign-in is `POST /v1/auth/otp/request`, `POST /v1/auth/otp/verify`, `POST /v1/auth/refresh` and `POST /v1/auth/sign-out`. Only the first three are public.
- **Access token:** signed with HMAC-SHA256, valid 15 minutes, checked without a database read. A signed-out device's access token still works until it expires (at most 15 minutes).
- **Refresh token:** random secret, valid 60 days, works once. Only its SHA-256 is stored. Reusing an old one ends the session.
- **Secrets:** `AUTH_SECRET` (32 characters or more) is the one master secret. The token key and the phone-hash key are derived from it, so they never share a key. It is required outside dev. In dev a random secret is made at start-up, which means tokens stop working on restart and are not shared between Cloud Run instances.
- **Phone numbers:** never stored or logged. The account index holds a keyed HMAC of the number (see [data-model.md](data-model.md)).
- **Rate limits:** 5 code requests and 10 code checks per number per 15 minutes, held in memory per instance for now.

## To document here

- Data export and account deletion.
