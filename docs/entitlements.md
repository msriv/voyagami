# Entitlements

Every plan limit is decided on the server by `services/api/internal/entitlements`. The app only reflects the answer. The numbers come from the Plans Spec board.

## Using it

```go
svc := entitlements.New(entitlements.Free())
d := svc.Check(user, trip, entitlements.BestOrder)
// d.Allowed, d.Limit (-1 is unlimited), d.Used, d.NeedsPlus
```

- `Check` asks about adding one more of a counted feature (`used < limit`), or using a flag feature. Pass `nil` for the trip when the check is not about one trip.
- `User.Plus` is true for an active subscription or trial. `User.Used` holds the person's counters (`UpcomingTrips`, `AskDrafts`).
- `Trip.Pass` is an active Trip Pass. `Trip.MemberPlus` is true when any member has Plus. `Trip.Used` holds the trip's counters (`BestOrder`, `ForwardedBookings`, `Files`, `JournalPhotos`).
- Unknown features are refused.

## Rules

- Free values: 2 upcoming trips, 1 best order per trip, 5 forwarded bookings, 3 files, 30 photos, 3 Ask drafts a month. Multi-currency, live crew map, receipt scans, offline maps, Gmail import and dark mode are Plus only.
- Whole-trip features (best order, bookings, files, multi-currency, live crew map) unlock for everyone on a trip when any member has Plus or the trip has a Trip Pass.
- A Trip Pass opens trip features for the crew but never the account ones: upcoming-trip count, Ask drafts, dark mode.
- Always free, whatever the table says: joining trips, settle-up, meet points, data export, offline plan and tickets.
- Every environment uses the Free table. In dev, an account with no mock plan gets `User.Unlimited` and skips it; the three mock accounts get Free, Plus and Trip Pass (see `limits.md`).
- Downgrading never deletes data. When a person or trip is over a cap (for example after Plus ends), `Decision.ReadOnly` is true: the content stays and can be read, but it cannot be changed until the count is back under the cap or Plus returns. Being exactly at the cap is full, not read-only.

## Endpoints and app hook

- `GET /v1/me/entitlements` answers every feature for the caller with no trip in view.
- `GET /v1/trips/{tripId}/entitlements` answers for one trip, so a Trip Pass or a crew member's Plus counts. Non-members get 404.
- Both return `{plan, features}` where each feature is `{allowed, limit, used, needsPlus}`. They answer 503 if plans are not wired.
- Plan facts come from the server-owned `entitlements/{uid}` Firestore document (`plus.active`, a future `trial.endsAt`, `usage.askDrafts`). No document means Free.
- Trip Passes and per-trip counters have no storage yet, so they read as none and zero. Dev accounts without a mock plan have no limits anyway.
- In the app, `useEntitlement(feature, tripId?)` reflects the answer. It allows nothing until the server answers or if the request fails.

`DEV_PLAN=free` or `DEV_PLAN=plus` (dev only) forces every account onto that plan, mock accounts included.
