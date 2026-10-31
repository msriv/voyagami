# Performance budgets

VOY-73 sets budgets for the app. CI can check some without a device. The rest need a phone and a person.

## Checked in CI

### Bundle size (stands in for cold start)

`scripts/check-bundle-size.sh` exports the Android app, sums the Hermes bytecode files and fails when the total passes `BUDGET_BYTES`. The budget is 5 MiB. The baseline on 2026-10-31 was 4.05 MB.

A bigger bundle loads slower, so this catches the most common cause of a slow cold start. It does not measure cold start itself.

To raise the budget, change the default in the script in a reviewed change and say why in the PR.

Run it locally: `scripts/check-bundle-size.sh`. Set `BUDGET_BYTES` to try another limit.

## Needs a device (not checked in CI)

These stay open for a person with a phone. Check them during dogfooding.

- Cold start under 2.5 seconds on a mid-range Android phone.
- 60 frames per second while scrolling an itinerary with 200 stops.
- Battery use with live location sharing on, over a one-hour walk.
- Image sizes and caching: photos load at the size shown, and a second view comes from cache.

## Not built

- A latency test for the API (p95 under 400 ms on the hot routes). The API runs on Cloud Run with Firestore, so a useful number needs the deployed service, not in-memory fakes. Measure it with the uptime check once staging exists (P1).
