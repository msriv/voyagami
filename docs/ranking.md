# Ranking

`services/api/internal/ranking` scores places for "Picks for you" (VOY-181). It is a pure function with no I/O. The recommendations endpoint (VOY-182) feeds it candidates and shows the result.

## What goes in

A `Place` holds what we know (tags, coordinates, rating and review count, price level, crew saves, opening hours, how complete the record is). A `Context` holds the crew's style, an anchor point (the stay or the day's stops) and the planned minute of the day. Zero values mean unknown. There is no field for commission or for whether a place pays us, so neither can change the order.

## Score

Each signal is 0 to 1. The score is their weighted sum, so it is also 0 to 1.

| Signal               | Weight | How it is worked out                                                                                                          |
| -------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------- |
| Fit to the crew      | 0.25   | Share of the crew's interests the place suits, nudged by how close its price is to their spending. Unknown style gives 0.5.   |
| Quality              | 0.20   | Rating pulled towards 3.8 by a prior worth 20 reviews, so one 5-star review loses to 800 reviews at 4.6. No rating gives 0.5. |
| Distance             | 0.20   | `1 / (1 + metres / 2000)` from the anchor. Unknown gives 0.5.                                                                 |
| Crew saves, must-see | 0.15   | Log scale of saves (full at 100), plus 0.5 for a curated must-see.                                                            |
| Time of day          | 0.10   | 1 if open at the planned minute, 0 if closed, 0.5 if unknown. Places known to be closed are dropped.                          |
| Data quality         | 0.10   | How complete the record is.                                                                                                   |

The weights are our starting point; the Recs Spec board names the signals but not the numbers. `TestGolden` pins the order for a small Hanoi list. If a change moves it, run `go test ./internal/ranking -update`, read the diff and keep it only if the new order is the one you want.

## Variety

After scoring, picks are chosen one at a time. Each pick's score is multiplied by 0.85 for every earlier pick of the same kind, so five temples in a row become a temple, a market, then the next temple. Ties break by ID, so the order is the same on every run.

## Reasons

Each result carries up to two reason codes: fit to the crew, close to the anchor, well rated (at least 50 reviews and about 4.2 stars), or saved by many crews. "Saved by crews" only counts from 20 crews (`CrewMinimum`), as the board says. The endpoint turns codes into plain text through i18n; the model never adds a fact.

## Not here

Choosing candidates (the trip's cities, opening on its dates, inside the budget), weather, holidays, and the free plan's top 5 cut are done around this function, not inside it.

## The endpoint (VOY-182)

`GET /v1/trips/{tripId}/recommendations?category=eat` ranks catalogue places for a trip's crew. Members only.

- **Category** is one of our six groups: `see`, `eat`, `stay`, `do`, `shop`, `nightlife`.
- **Centre** is `lat` and `lng` when given, otherwise the middle of the trip's stops that have a position. With neither the answer is 400. The pool is the 200 nearest places in the group within `radiusMeters` (default 10 km, at most 20 km).
- **Style** is the union of the active members' interests and their average spend (shoestring 1, comfort 2, splurge 3). A place suits an interest when its `tags` contain the onboarding interest id (for example `foodTrails`), so the places pipeline must write tags in those words.
- **Not stored yet**, so they read as unknown: crew saves, opening hours and the must-see flag.
- **Reasons** come back as codes (`style`, `quality`, `distance`, `popularity`). The app turns them into plain words through i18n; the server writes no sentences.
- **hasMore** is true when more picks exist than `limit` (default 20, at most 50). The Free plan's top 5 cut is VOY-173.
- No sponsored items until P2.
