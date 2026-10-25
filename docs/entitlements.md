# Entitlements

Every plan limit is decided on the server by `services/api/internal/entitlements`. The app only reflects the answer. The numbers come from the Plans Spec board.

## Using it

```go
svc := entitlements.New(entitlements.ForEnv(cfg.Env))
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
- Dev has no limits (`ForEnv("dev")`). Staging and prod use the Free values.
- Downgrading never deletes data. A check that says no on content already over the limit means the content is read-only (VOY-115).

Next: the endpoint and app hook (VOY-114), then the crew-unlock and downgrade tests (VOY-115).
