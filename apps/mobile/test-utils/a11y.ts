import { Pressable, TouchableOpacity } from "react-native";
import { screen } from "expo-router/testing-library";

/**
 * Walks the rendered screen and lists every touchable that a screen reader could not use: no role,
 * or no name (an accessibility label, or text inside it). Touchables hidden from screen readers
 * are skipped. An empty list means the screen passes this part of docs/accessibility.md.
 */
export function inaccessibleControls(): string[] {
  const bad: string[] = [];
  for (const type of [Pressable, TouchableOpacity]) {
    for (const node of screen.UNSAFE_getAllByType(type, { hidden: true })) {
      let hidden = false;
      for (
        let n = node as { props: Record<string, unknown>; parent?: unknown } | null;
        n;
        n = n.parent as typeof n
      ) {
        if (
          n.props?.accessibilityElementsHidden ||
          n.props?.importantForAccessibility === "no-hide-descendants"
        )
          hidden = true;
      }
      if (hidden) continue;
      const p = node.props as Record<string, unknown>;
      const role = p.accessibilityRole ?? p.role;
      const label = p.accessibilityLabel ?? p["aria-label"];
      const hasText =
        JSON.stringify(node.props.children ?? "").includes('"type"') ||
        typeof node.props.children === "string";
      if (!role || !(label || hasText)) bad.push(`${String(role)} / ${String(label)}`);
    }
  }
  return bad;
}
