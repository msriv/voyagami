# Testing

## What runs where

- **App components and screens:** `pnpm --filter @voyagami/mobile test`. These render real screens with a mocked API (`test-utils/api.tsx`), so every screen has a test for its main flow and its error states.
- **Coverage:** `pnpm --filter @voyagami/mobile test:coverage` prints a summary. There is no percentage gate. Money, entitlements and the write queue must have tests, and review checks that they do.
- **Accessibility sweep:** `__tests__/a11ySweep.test.tsx` fails when a tappable control has no role or name (see `docs/accessibility.md`).
- **Go API:** table-driven unit tests in `services/api`, and the Firestore rules tests on the emulator (the "Firestore rules on the emulator" CI job).
- **API contract:** the server is generated from `api/openapi.yaml` and the app client is generated from the same file, so a change to one without the other fails typecheck.

## End-to-end flows (Maestro)

Flows live in `apps/mobile/.maestro/`. They need a dev build on a simulator or phone and the dev API, whose stub SMS provider accepts code `123456`.

```
maestro test apps/mobile/.maestro/sign-in.yaml
```

`sign-in.yaml` is the smoke flow. The flows for create trip, add place, invite, add expense and settle up are still to write, and none of them has run in CI yet: that needs a simulator runner, which is not set up. Until then, run them by hand before a release.
