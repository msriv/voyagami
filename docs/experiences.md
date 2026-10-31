# Experiences

`GET /v1/trips/{tripId}/experiences` lists things to do near the trip (VOY-47). It works like Stays.

## How it works

- Places come from our own catalogue: the "do" group, within 10 km of the middle of the trip's stops that have a position, nearest first, at most 20. With no such stop the list is empty.
- Each place gets one hand-off link per partner, written with the same builder as Stays (`stays.Build`). The search term is the place name. Nothing is booked or paid here.
- `POST .../experiences/clicks` counts a tap per partner and day, in the same store as the stay clicks. Experience partner ids are saved with an `exp-` prefix so the two never mix. No person, trip or place is kept.
- To put an experience in the itinerary, the app adds a stop of kind `catalog` with the `placeId` and the time the person picks. No new route is needed.

## Partners

`EXPERIENCE_PARTNERS` is a JSON list with the same shape as `STAY_PARTNERS` (see `docs/stays.md`). With nothing set, places still list but `links` is empty. Add GetYourGuide, Viator or Klook here once their affiliate accounts exist (needs Mihir). No partner name or ID is written in code.

## Not done

- Partner inventory (real tours, prices, availability) is not fetched. The list is catalogue places only, until partner accounts and their APIs exist.
- The list is not filtered by the trip's dates.
