import { recapCards } from "./recapCards";

const base = {
  tripId: "t",
  days: 0,
  stops: 0,
  crew: 1,
  distanceMeters: 0,
  spentMinor: 0,
  currency: "VND",
  highlights: [],
};

describe("recapCards", () => {
  it("is just a cover and a closing for an empty trip", () => {
    expect(recapCards(base).map((c) => c.kind)).toEqual(["cover", "closing"]);
  });

  it("adds a card for each thing there is to say, highlights last", () => {
    const cards = recapCards({
      ...base,
      days: 3,
      spentMinor: 5,
      favouriteDay: { date: "2026-01-02", entries: 1 },
      highlights: [
        { date: "2026-01-02", authorUid: "a", text: "x" },
        { date: "2026-01-03", authorUid: "a", text: "y" },
      ],
    });
    expect(cards.map((c) => c.kind)).toEqual([
      "cover",
      "stats",
      "money",
      "day",
      "highlight",
      "highlight",
      "closing",
    ]);
    expect(cards[5]).toEqual({ kind: "highlight", index: 1 });
  });
});
