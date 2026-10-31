import { screen } from "expo-router/testing-library";

interface Node {
  type: unknown;
  props: Record<string, unknown>;
  parent: Node | null;
  children: (Node | string)[];
}

function all(node: Node, pred: (n: Node) => boolean, out: Node[] = []): Node[] {
  if (pred(node)) out.push(node);
  for (const c of node.children) if (typeof c !== "string") all(c, pred, out);
  return out;
}

/**
 * Walks the rendered screen and lists every touchable that a screen reader could not use: no role,
 * or no name (an accessibility label, or text inside it). Touchables hidden from screen readers
 * are skipped. An empty list means the screen passes this part of docs/accessibility.md.
 */
export function inaccessibleControls(): string[] {
  const root = screen.root as unknown as Node;
  const bad: string[] = [];
  for (const node of all(root, (n) => typeof n.props.onResponderGrant === "function")) {
    let hidden = false;
    for (let n: Node | null = node; n; n = n.parent) {
      if (
        n.props.accessibilityElementsHidden ||
        n.props.importantForAccessibility === "no-hide-descendants"
      )
        hidden = true;
    }
    if (hidden) continue;
    const p = node.props;
    const role = p.accessibilityRole ?? p.role;
    const label = p.accessibilityLabel ?? p["aria-label"];
    const hasText = all(node, (n) => n.type === "Text").length > 0;
    if (!role || !(label || hasText)) bad.push(`role=${String(role)} label=${String(label)}`);
  }
  return bad;
}
