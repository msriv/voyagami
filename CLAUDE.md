# Voyagami: instructions for Claude Code

Voyagami is a mobile-only group travel planner and journal (domestic and international). iOS and Android only. No website.

The work is tracked in Linear, team **VOY** (https://linear.app/voyagami). The design lives in `design/`. Read this file at the start of every session.

## Phases (read this first)

Mihir is the only developer. The work is dev-first: build a complete app on a dev setup, use it on a real trip, then harden it for production.

| Phase | Linear project | Goal |
|---|---|---|
| D0 | Setup & walking skeleton | Repo, CI, dev GCP project, API + app skeleton, stub sign-in. End: sign in on a simulator against the dev API |
| D1 | Solo planner | Onboarding, locale, trips, itinerary, add place, best order, map, entitlements service (all unlimited) |
| D2 | Crew & money | Bookings, invites, live co-editing, roles, polls, meet points, expenses, multi-currency, settle-up, push |
| D3 | Journal, offline & recap | Journal, photos, recap player, local-first storage, sync status |
| D4 | Plans, discovery & polish | Places pipeline (one pilot city), ranking, recs, stays, RevenueCat sandbox, plans, paywall, limits, tour, i18n. Ends with the **Dogfood gate** |
| P1 | Production foundation | Staging/prod, CD, observability, real SMS, security, privacy, a11y audit, tests, backups |
| P2 | Launch | Stores, legal, beta, analytics, recap video, Ask, sponsored cards, offline map packs |
| P3 | Post-launch | Cloud SQL + PowerSync plan, flight alerts |

Rules:
- Work phases in order. Do not start a P-phase ticket until the Dogfood gate ticket is Done.
- D-phase uses **dev-only infrastructure**: one GCP project (`voyagami-dev`), emulators, stubs for SMS (code 123456), payments (RevenueCat sandbox, D4 only) and partners. Do not create staging/prod, Sentry, store listings or paid third-party accounts.
- All limits are unlimited in dev until the D4 limits epic; the entitlements service exists from D1. Never skip the service.
- Places data covers one pilot city (Hanoi) until P1.
- Epics (label `epic`) are containers. Never implement an epic directly; work its sub-issues in order (1/3, 2/3, ...). P-phase tickets are coarse on purpose: when one is next, split it into 1 to 3 point sub-issues first, then work those.

## How to pick up work

1. Use the Linear tools to list **sub-issues and non-epic issues** in team VOY with label `agent-ready`, status `Todo`, and no unresolved blockers, in the lowest phase project first (D0 < D1 < ... < P3). If nothing is `Todo`, take the next `Backlog` sub-issue (lowest number) of the current phase whose epic's blockers are all Done, and move it to Todo yourself. Ties go to higher priority.
2. Skip anything labelled `needs-human` (accounts, keys, legal, pricing decisions). If your ticket is blocked by one, stop and tell Mihir what is needed.
3. Move the ticket to **In Progress**. Create a branch with the name Linear suggests (`gitBranchName`). Linear mirrors issues to GitHub issues; treat Linear as the source of truth and ignore the GitHub copies.
4. Read the ticket fully, then open the board(s) it names in `design/boards/`. See `design/BOARD_INDEX.md` for which file is which.
5. Build it. Meet every acceptance criterion. If something is ambiguous, check the matching spec board (Plans Spec, Recs Spec, Locale Spec, Tech Stack, Data Map) before asking.
6. Open a PR titled `VOY-123: <ticket title>`, linking the ticket. Put a short "How I tested this" and screenshots for any UI change. Move the ticket to **In Review**.
7. Merge only after Mihir says so. Claude's GitHub access acts as Mihir, so GitHub's Approve button is not available on Claude's PRs. Instead, Mihir comments "LGTM" on the PR (or says so in the project thread). Once he has, every review thread is resolved and CI is green on the latest commit, merge with a merge commit. A push after his LGTM needs a fresh LGTM.
8. After merging, check the ticket moved to Done (the Linear integration does this from `Closes VOY-123`); if not, move it yourself.
9. If you find work that is not covered by a ticket, create a new one (label it, link it to the project) rather than quietly widening the PR.

One ticket per PR. Keep PRs small. Never commit secrets.

## Locked stack

- **App:** Expo (React Native), TypeScript strict, Expo Router
- **API:** Go on Cloud Run. OpenAPI 3.1 is the source of truth. The API is Go, not TypeScript.
- **Data:** Firestore at launch (later Cloud SQL + PowerSync, see the migration ticket). Pub/Sub for jobs.
- **AI:** Gemini on Vertex, only through the backend
- **Auth:** phone OTP (MSG91 for India behind a provider interface; another provider elsewhere), FCM/APNs for push
- **Payments:** RevenueCat over the App Store and Google Play. We take no payments ourselves.
- **Maps:** Google Maps SDK online. Mapbox with OpenStreetMap-based data **only** for offline packs. Never draw Google Places data on Mapbox.
- **Infra:** Terraform, GitHub Actions, workload identity federation (no long-lived cloud keys)
- **Errors/monitoring:** Sentry, Cloud Logging, uptime checks

## Rules that must never be broken

**Nothing is hard-coded to one country.**
- No hard-coded currency symbols, date formats, units, phone formats or payment apps. Use the locale module (VOY-58).
- The mocks use a Bengaluru persona with ₹, km and UPI. That is sample data. Prices in the mocks (₹999, $29.99, ₹199/349/499) are illustrative; real prices come from the stores.
- Four sources of truth: the user (home region, units), the trip (settlement currency, timezone), the destination (local currency, language), the app store (prices).

**Money.**
- Store amounts as integer minor units plus an ISO-4217 currency code. Never floats.
- Currency decimals come from ISO data (JPY and VND have 0).
- Each trip has one settlement currency. Each expense stores its original amount, currency and the rate used.

**Plans and limits.**
- Every limit goes through the entitlements service (VOY-51) on the **server**. The app only reflects the answer.
- Always free: joining trips, settle-up, meet points, data export, offline plan and tickets.
- Whole-trip features unlock for the whole crew if anyone has Plus or the trip has a Trip Pass.
- Downgrading never deletes data. Over-limit content becomes read-only.

**Copy.**
- Plain English only. No foreign-language words in the UI (the Japanese influence is in the look only).
- All user-facing strings go through i18n.

**Accessibility.**
- Every tappable thing has at least a 44x44 hit area (use `hitSlop`), a role, a label and a state.
- Controls are real buttons, never plain text.
- Text must scale to 200% without clipping on a 320px-wide screen.
- Honour reduce-motion.

**Privacy.**
- No phone numbers, exact locations or free text in logs or analytics.
- Location sharing is opt-in and ends automatically.
- Only strip or keep photo EXIF location as the user chooses.

**Data sources and licences.**
- Places base: Foursquare Open Source Places (Apache 2.0), OpenStreetMap, Wikidata. Keep source and licence per field.
- Google Places data is fetched live for display only, not stored beyond the allowed cache window.
- Photos need their attribution shown. Credits are listed on the Recs Spec board.

**Sponsored cards.** At most one per list, labelled "Sponsored", hideable for 30 days, shown on **all** plans including Plus.

## Design system

Tokens are in `design/tokens.json`. Short version:

| Token | Value | Notes |
|---|---|---|
| indigo | #2D3E8C | primary |
| vermilion | #E5482F | accent; use for ink-on-light text only where contrast passes |
| washi | #F4F0E6 | app background |
| ink | #14161B | text |
| matcha | #D3E4C5 | success / soft fill |
| gold | #FFE27A | highlight |
| muted | #5B6070 | secondary text |

Fonts: Bricolage Grotesque (display), Manrope (body), Instrument Serif italic (accent words), Caveat (journal handwriting).
Motifs: wave pattern, paper-plane seal, circle brush mark, washi tape, folded corner, airmail stripe, postmark.

The design boards are HTML files. They are **reference, not code to copy**: rebuild each screen with React Native components from the shared UI kit (VOY-12). Match spacing, type and behaviour; do not paste inline styles.

## Repo layout

```
apps/mobile      Expo app
services/api     Go API
packages/        shared types, generated api-client
infra/           Terraform (one root per environment)
design/          boards, photos, tokens (read-only reference)
docs/            data-model.md, sync.md, security.md, ...
CLAUDE.md        this file
```

## Commands (set up by VOY-5)

`make setup`, `make dev`, `make test`, `make lint`, `make api-client`.
Run `make lint test` before every PR.

## Definition of done for a ticket

- All acceptance criteria met, each with a test where it can be tested
- Lint, typecheck and tests pass in CI
- Works offline where the ticket says so
- Checked with a screen reader label pass and at 200% text size for any UI
- Docs updated if behaviour or data shape changed
- No hard-coded locale values (grep passes)
