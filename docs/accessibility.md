# Accessibility

> The shared UI kit (VOY-12) builds these rules in; P1 runs a full audit. Every UI PR is checked against this list.

## UI kit (`apps/mobile/src/ui`)

- `Text` always scales with the system text size; it has no prop to turn scaling off or cap it. Use `heading` for headings.
- `Button` and `IconButton` always set role `button`, a label and state (`disabled`, `busy`). They are at least 44x44 with `hitSlop` around that. `IconButton` requires `accessibilityLabel`.
- Pressed feedback is dimming plus a small scale; the scale is dropped when reduce motion is on (`useReducedMotion`).
- Accent (vermilion) buttons use ink text, never white, so contrast passes.
- Tests for these props live in `apps/mobile/src/ui/__tests__`. They cannot show 200% text or screen reader order; check those on a device.

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
