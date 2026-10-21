# apps/mobile

The Expo (React Native) app: Expo Router, strict TypeScript, Jest.

```
pnpm --filter @voyagami/mobile start     # Metro; press i (iOS simulator) or a (Android emulator)
pnpm --filter @voyagami/mobile test
pnpm --filter @voyagami/mobile typecheck
```

Routes live in `app/`. The tab shell (Home, Explore, Profile) is in `app/(tabs)`. Theme, fonts and
i18n come in VOY-99; the development build and device guide in VOY-100.
