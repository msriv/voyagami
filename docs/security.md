# Security

> Stub. Grows with the D0 auth tickets and is hardened in P1. Report anything that looks wrong to Mihir rather than in a public issue.

## Secrets

- Never commit secrets. `.env` files, keys, service-account JSON and Firebase config files are git-ignored; commit an `.env.example` with placeholders instead.
- CI and Terraform reach Google Cloud through workload identity federation. No long-lived cloud keys.
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

## To document here

- Token lifetimes, refresh and sign-out.
- Data export and account deletion.
