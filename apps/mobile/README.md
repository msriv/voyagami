# apps/mobile

The Expo (React Native) app: Expo Router, strict TypeScript, Jest.

```
pnpm --filter @voyagami/mobile start     # Metro; press i (iOS simulator) or a (Android emulator)
pnpm --filter @voyagami/mobile test
pnpm --filter @voyagami/mobile typecheck
```

Routes live in `app/`. The tab shell (Home, Explore, Profile) is in `app/(tabs)`. Theme, fonts and
i18n come in VOY-99; the development build and device guide in VOY-100.

## Theme, fonts and strings

- `src/theme`: typed theme built from `design/tokens.json` (`useTheme()`), and the font families (Bricolage Grotesque, Manrope, Instrument Serif italic, Caveat) loaded at start-up in `app/_layout.tsx`. Dark tokens are stubbed to match light until they are designed.
- `src/i18n`: English strings in `src/i18n/en.ts`, read with `useTranslation()`. ESLint (`react/jsx-no-literals`) fails on literal text in JSX, so every user-facing string goes through i18n.
- Text scaling: never set `allowFontScaling={false}` or cap `maxFontSizeMultiplier`; the app must stay usable at 200%.

## UI kit

`src/ui` holds the shared components (`Text`, `Button`, `IconButton`; more in VOY-102 and VOY-103). Use them instead of raw `Text` and `Pressable` so role, label, state and the 44 point hit area are never forgotten. See [docs/accessibility.md](../../docs/accessibility.md).
