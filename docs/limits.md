# Plan limits on the server

Every limit is checked on the server by the entitlements service. The app only reflects the answer.

## What is enforced

| Feature             | Where                                         | Counted how                                                                                      |
| ------------------- | --------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `upcomingTrips`     | create trip                                   | trips the person owns that are planned and not yet ended. Joining a trip is free and not counted |
| `bestOrder`         | suggest best order                            | `usage.bestOrderRuns` on the trip, raised after each run                                         |
| `forwardedBookings` | import bookings                               | `usage.forwardedBookings` on the trip, raised by the number of bookings found                    |
| `multiCurrency`     | add or change an expense in a second currency | a flag, no counter                                                                               |

Files, journal photos and top picks are limited when their endpoints arrive (signed uploads, photo pipeline, recommendations).

Plus for any member, or a Trip Pass, lifts the whole-trip limits for everyone on the trip. Plus on the person lifts the person limits.

## What the app sees

A refused request answers `403` with `code: "limit_reached"` and `feature` set to the feature name (the same names as `GET /v1/me/entitlements`). The app shows the matching paywall moment. The numbers (`limit`, `used`, `needsPlus`, `readOnly`) come from the entitlements routes.

## Dev

Dev runs with no limits. Set `DEV_PLAN=free` on the API to try the Free limits with real counters, or `DEV_PLAN=plus` to force Plus.
