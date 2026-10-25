import type { components } from "@voyagami/api-client";

type Recap = components["schemas"]["Recap"];

export type RecapCard =
  | { kind: "cover" }
  | { kind: "stats" }
  | { kind: "money" }
  | { kind: "day" }
  | { kind: "highlight"; index: number }
  | { kind: "closing" };

/** The cards of the story, in order. Cards with nothing to say are left out. */
export function recapCards(recap: Recap): RecapCard[] {
  const cards: RecapCard[] = [{ kind: "cover" }];
  if (recap.days > 0 || recap.stops > 0) cards.push({ kind: "stats" });
  if (recap.spentMinor > 0) cards.push({ kind: "money" });
  if (recap.favouriteDay) cards.push({ kind: "day" });
  recap.highlights.forEach((_, index) => cards.push({ kind: "highlight", index }));
  cards.push({ kind: "closing" });
  return cards;
}
