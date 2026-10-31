# Plan limits on the server

Every limit is checked on the server by the entitlements service. The app only reflects the answer.

## What is enforced

| Feature             | Where                                         | Counted how                                                                                      |
| ------------------- | --------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `upcomingTrips`     | create trip                                   | trips the person owns that are planned and not yet ended. Joining a trip is free and not counted |
| `bestOrder`         | suggest best order                            | `usage.bestOrderRuns` on the trip, raised after each run                                         |
| `forwardedBookings` | import bookings                               | `usage.forwardedBookings` on the trip, raised by the number of bookings found                    |
| `multiCurrency`     | add or change an expense in a second currency | a flag, no counter                                                                               |
| `topPicks`          | recommendations list                          | a cap on the list: Free sees the top 5 per category, the response says `hasMore`; no counter     |

## Monthly counters and flags (answers ready, features come later)

- **Ask drafts**: 3 a month on Free. `entitlements/{uid}.usage` holds `periodStart` (the first instant of the month, UTC) and `askDrafts`. A counter from an earlier month reads as zero, and the first draft of a new month starts it again, so nothing has to run at midnight. `AddAskDraft` counts one in a transaction. Ask itself ships in P2.
- **Recap length** (`recapSeconds`): Free recaps are capped at 30 seconds and carry the watermark. Plus on anyone in the crew, or a Trip Pass, gives full length with no watermark. The recap renderer asks `Service.Recap(user, trip)` for `{ MaxSeconds, Watermark }`. In `GET .../entitlements` the feature's `limit` is the seconds (`-1` for no cap) and `used` is always 0.
- **Offline maps**: closed on Free, open with the person's own Plus (a crew member's Plus does not open them). The check already existed; the feature ships in P2.

Files and journal photos are limited when their endpoints arrive (signed uploads, photo pipeline). The checks already exist in the entitlements service (`files`, `journalPhotos`: 3 files and 30 photos per trip on Free).

Top picks (`GET /v1/trips/{id}/recommendations`) belong to the person: their own Plus or a Trip Pass on the trip shows the full list; a crew member's Plus does not.

Plus for any member, or a Trip Pass, lifts the whole-trip limits for everyone on the trip. Plus on the person lifts the person limits.

## What the app sees

A refused request answers `403` with `code: "limit_reached"` and `feature` set to the feature name (the same names as `GET /v1/me/entitlements`). The app shows the matching paywall moment. The numbers (`limit`, `used`, `needsPlus`, `readOnly`) come from the entitlements routes.

## Dev

Dev gives each account its own plan. An ordinary dev account has no limits, so the owner can test freely. Three mock accounts stand for one plan each. Sign in with the number and the dev code `123456`:

| Number          | Plan                                                                                                         |
| --------------- | ------------------------------------------------------------------------------------------------------------ |
| +1 555 555 0101 | Free, with the real Free limits and counters                                                                 |
| +1 555 555 0102 | Plus                                                                                                         |
| +1 555 555 0103 | Trip Pass: this account is Free, but every trip it is a member of has a Pass, so the crew sees the Pass plan |

The mark is stored on the account (`entitlements/{uid}.devPlan`) and set again at every sign-in. These numbers are in the fictional 555-01xx range and only work when `APP_ENV` is dev. `DEV_PLAN=free` or `DEV_PLAN=plus` still forces every account onto one plan.
