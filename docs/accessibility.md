# Accessibility

> The shared UI kit (VOY-12) builds these rules in; P1 runs a full audit. Every UI PR is checked against this list.

## UI kit (`apps/mobile/src/ui`)

- `Text` always scales with the system text size; it has no prop to turn scaling off or cap it. Use `heading` for headings.
- `Button` and `IconButton` always set role `button`, a label and state (`disabled`, `busy`). They are at least 44x44 with `hitSlop` around that. `IconButton` requires `accessibilityLabel`.
- Pressed feedback is dimming plus a small scale; the scale is dropped when reduce motion is on (`useReducedMotion`).
- Accent (vermilion) buttons use ink text, never white, so contrast passes.
- `Chip` is a button with `selected` and `disabled` state; selection is also shown by fill and weight, not colour alone.
- `Input` shows its label above the field (not as a placeholder), sets it as the accessible label, and shows errors in a polite live region with a vermilion border plus text.
- `Sheet` is a modal with a heading, a close button and a tappable backdrop, both labelled; content scrolls so it stays reachable at 200% text. It has no slide animation when reduce motion is on.
- `Toast` is announced as an alert, lasts 5 seconds by default and always has a labelled dismiss button.
- Labels for close and dismiss are props, so the caller passes them through i18n.
- Motifs (`Seal`, `WavePattern`, `BrushCircle`, `WashiTape`, `Postmark`) are decoration: hidden from screen readers unless given a label. `Postmark` reads as one phrase, "city, date".
- Tests for these props live in `apps/mobile/src/ui/__tests__`. They cannot show 200% text or screen reader order; check those on a device.

## Automatic checks

- ESLint (`eslint-plugin-react-native-a11y`) fails the build on invalid roles, states, live regions, nested touchables and touchables with no role or label. Run by `make lint`.
- `apps/mobile/__tests__/a11ySweep.test.tsx` renders the main screens and fails if any touchable has no role or no name (`test-utils/a11y.ts`). Add a screen to its list when you add one.
- These cannot judge reading order, 200% text or contrast. Do the checklist below on a device before each release.

## Release checklist

Run this on one iPhone and one Android phone before each release and tick it in the release PR.

1. VoiceOver / TalkBack: walk Home, a trip, Money, Journal, Plans and Profile. Order is logical; each control reads name, role and state.
2. Largest system text size on a 320px-wide device: nothing clips or overlaps, on every screen above and the paywall sheet.
3. Contrast: body text 4.5:1, large text 3:1; vermilion only on ink-on-light accents.
4. Reduce motion on: the recap, tour and sheets do not animate.
5. Tap targets: every control is at least 44x44.

## Rules

- Every tappable thing has at least a 44x44 hit area (use `hitSlop`), an accessibility role, a label and a state.
- Controls are real buttons, never plain text.
- Text scales to 200% without clipping on a 320px-wide screen.
- Honour reduce-motion: no auto-playing or decorative motion when it is on.
- Colour contrast passes WCAG AA. Vermilion text on light backgrounds only where contrast passes.
- All user-facing strings, including accessibility labels, go through i18n.

## How to check a UI change

1. Turn on VoiceOver (iOS) or TalkBack (Android) and move through the screen. Every control reads a sensible label, role and state, in a sensible order.
2. Set the largest text size (200%) on a small device or a 320px-wide simulator. Nothing clips or overlaps.
3. Turn on reduce-motion and check that animations are replaced or removed.
4. Add screenshots at 200% text size to the PR.
