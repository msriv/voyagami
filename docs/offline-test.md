# Airplane mode test script

Run this on a phone (or simulator) with the dev API. About ten minutes. It checks that a trip can be used with no network, and that changes made offline arrive later without doubling.

## Before

1. Sign in (dev code 123456) and open a trip that has at least three places, one booking and one expense.
2. Open each of these once while online: Home, the trip, Itinerary, Bookings, Money, Journal. Opening a screen is what saves it for offline use.

## Offline reads

3. Turn on airplane mode.
4. Force-quit and reopen the app. You should still be signed in.
5. Home lists your trips. Open the trip. Itinerary, Bookings, Money and Journal all show what you saw before. The sync badge in the trip header says `No signal`.
6. A screen you never opened online shows a plain error, not a crash. That is expected.

## Offline writes

7. Add an expense (Money, Add expense). It saves. The header badge says `No signal. 1 change saved on your phone`.
8. Tap the badge. `Changes to send` lists the expense with Send now and Discard.
9. Turn airplane mode off. Within about 15 seconds the badge says `Everything is saved` and the expense is in the list once, not twice.
10. Repeat 7 and 9 with two expenses. They arrive in the order made.

## Saved on this phone

11. Profile, `Saved on this phone` shows a size. Press `Clear saved copies`. The size drops.
12. Turn airplane mode on and open the trip: it shows an error (copies are gone) until you go online again. Change 7 must survive clearing: add an expense offline, clear, and it is still listed under `Changes to send`.

## Not covered yet

Ticket files attached to bookings are not downloaded for offline use yet; file uploads are not built (VOY-131, VOY-161).
