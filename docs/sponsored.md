# Sponsored cards

Direct deals only. No ad network, no tracking of the person.

## Rules

- At most one card per list. The API returns one card or none.
- The same on every plan, Plus included.
- The app always draws the "Sponsored" label. The card's `sponsor` is shown beside it.
- A person can hide a card. It stays away for 30 days (`model.HideSponsoredFor`).
- Targeting is by destination country and place group only.

## Inventory

Collection `sponsoredCards`, filled by hand in the Firebase console. Fields: `active` (bool), `sponsor`, `title`, `body`, `imageUrl`, `url`, `countries` (ISO 3166-1 alpha-2, empty means all), `groups` (place groups, empty means any list), `priority` (higher wins), `startsAt`, `endsAt` (empty means no end).

When several cards fit, the highest priority wins, then the lowest ID.

## Counting

`POST /v1/sponsored/{cardId}/impressions` and `/clicks` count once per person, card, kind and day. Totals sit in `sponsoredStats/{cardId}_{day}` as `impressions` and `clicks`. The dedupe marker lives under `users/{uid}/sponsoredSeen`, so deleting the account removes it. No location or free text is stored.

Hidden cards live in `users/{uid}/hiddenCards`.

## Limits

The route reads every active card and filters in memory. Fine for a few dozen deals.
