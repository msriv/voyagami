# Data model

Firestore at launch, in the `voyagami-dev` project during the D phases. A later move to Cloud SQL and PowerSync is planned (P3), so every shape below maps cleanly to a table. This page is the contract for the Go repository layer (VOY-96) and the security rules and indexes (VOY-97). Sources: the Data Map, Tech Stack, Locale Spec and Plans Spec boards (see [BOARD_INDEX](../design/BOARD_INDEX.md)). If this page and a spec board disagree, the board wins until this page is updated.

## Contents

- [Conventions](#conventions)
- [Value types](#value-types)
- [Collection map](#collection-map)
- [Roles and access](#roles-and-access)
- [Collections](#collections)
- [Indexes](#indexes)
- [Offline sync](#offline-sync)
- [Moving to Cloud SQL](#moving-to-cloud-sql)
- [Decisions to confirm](#decisions-to-confirm)

## Conventions

- **Trip membership drives all access.** Everything a trip owns lives under `trips/{tripId}/…` and carries `tripId`. Access is decided from the member document at `trips/{tripId}/members/{uid}`, never from a field on the trip.
- **Two writers.** Content that must work offline and merge freely (stops, notes, comments, votes, tasks, expenses, journal) is written by the app straight to Firestore, within the rules. Anything that is gated by a plan limit, needs one consistent answer, or touches permissions or money settlement is written only by the Go API (`server` in the access tables). Rules deny client writes to server-owned collections and fields. **Today the rules deny all direct client access** (VOY-97) and the API does every read and write; collections are opened to the app one at a time by the tickets that need them (D2 co-editing, D3 local-first sync), each with its own rule and tests. See [security.md](security.md). Limits are always checked on the server (CLAUDE.md), so every create that a limit can stop goes through the API.
- **IDs** are strings. The app makes the ID for content it creates (UUID v7), so a retried write cannot create a duplicate. The server makes IDs for server-owned documents. User IDs are our own stable account ID, not the phone number.
- **Every document** has `createdAt`, `updatedAt` (UTC instants, set by the server clock), `createdBy` and `updatedBy` (user IDs). They are left out of the tables below.
- **Instants** (booking times, `createdAt`) are UTC. **Calendar dates** the person thinks of as "21 Nov" are `YYYY-MM-DD` strings, and **local clock times** are `HH:mm` strings, each paired with an IANA time zone (`Asia/Ho_Chi_Minh`). Display follows the trip's time zone, then the viewer's formats.
- **Deleting** is a real delete except where noted (`deletedAt` tombstone) so that Restore in the activity feed works.
- **No arrays that grow without bound.** Small bounded lists (poll options, destinations, expense shares) are arrays; anything that can grow is a subcollection. Each such array becomes a child table later.
- **Nothing is hard-coded to a country.** No document stores a currency symbol, a date format, a unit label or a payment app name. Fields hold codes (ISO 3166-1 alpha-2 country, ISO 4217 currency, BCP 47 language, IANA zone) and the locale module (VOY-58) renders them.

## Value types

**Money** is the only way an amount is stored. Never a float.

```
Money { amountMinor: integer, currency: string }   // ISO 4217. 1000 with "VND" is 1000 dong; 1000 with "USD" is 10 dollars
```

Decimals come from ISO data (JPY and VND have 0, USD has 2, KWD has 3). Negative amounts are allowed only where a table says so.

**Rate** is a decimal string of how many settlement-currency units one original unit buys, for example `"0.003451"`. It is a string so no float touches it, and it is kept with its date and source:

```
Rate { value: string, asOf: string /* YYYY-MM-DD */, source: string }
```

**PlaceRef** points at a place without copying it:

```
PlaceRef = { kind: "catalog", placeId: string }
         | { kind: "custom", name: string, lat: number, lng: number }
```

A custom place is a pin the person dropped. A catalog place is a document in `places`.

**Provenance** records where a piece of places data came from, per field, so licences are honoured: `{ source, licence, sourceId, retrievedAt }`.

## Collection map

```
users/{uid}
  devices/{deviceId}
  stamps/{stampId}
phoneIndex/{phoneHash}
sessions/{sessionId}
entitlements/{uid}
invites/{inviteId}
places/{placeId}
trips/{tripId}
  members/{uid}
  days/{date}
  stops/{stopId}
  saved/{savedId}
  bookings/{bookingId}
  files/{fileId}
  polls/{pollId}
    votes/{uid}
  tasks/{taskId}
  comments/{commentId}
  meetPoints/{meetPointId}
  presence/{uid}
  expenses/{expenseId}
  settlements/{settlementId}
  journal/{entryId}
  media/{mediaId}
  passes/{passId}
  activity/{activityId}
```

Nesting is one level under `trips` (plus `votes` under a poll), which keeps rules short and each listener narrow, and keeps every table a flat `trip_id` child later.

## Roles and access

Roles are `owner`, `editor`, `viewer`, stored on the member document.

| Capability                                                                                                     | Owner  | Editor | Viewer |
| -------------------------------------------------------------------------------------------------------------- | :----: | :----: | :----: |
| Read everything in the trip (except other people's private journal entries and anything marked "server" below) |  yes   |  yes   |  yes   |
| Vote, comment, react, tick their own tasks, write their own journal, share their own location                  |  yes   |  yes   |  yes   |
| Edit plan, places, bookings, polls, tasks, expenses, meet points                                               |  yes   |  yes   |   no   |
| Change roles, remove people, create or revoke invites                                                          |  yes   |   no   |   no   |
| Close a poll, mark a settlement paid                                                                           | server | server | server |
| Delete the trip, change settlement currency                                                                    |  yes   |   no   |   no   |

A person removed from a trip loses access at once. Their queued offline writes are rejected on the next sync and undone on their phone, with a note saying why. Members can always leave a trip themselves. Joining a trip is always free (CLAUDE.md), so the join step never checks a plan.

Access to a trip's data checks that `trips/{tripId}/members/{uid}` exists and has `status == "active"`. Rules must never trust `memberUids` on the trip document, which only exists to list "my trips".

## Collections

Each section lists the path, fields, who can write, and what is worth knowing. "Offline" is covered in the [Offline sync](#offline-sync) table.

### `users/{uid}`

The person. Read and written by that person only (plus the server). Other people see the small copy held in each `members` document, never this one.

| Field                 | Type                                             | Notes                                                                                                                                                                                                                                            |
| --------------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `displayName`         | string                                           | Name the person chose. Optional until onboarding ends.                                                                                                                                                                                           |
| `photoUrl`            | string?                                          |                                                                                                                                                                                                                                                  |
| `homeRegion`          | string                                           | ISO 3166-1 alpha-2, for example `IN`. Set at onboarding and editable. Moving changes display only, never past trips. Saved with `PATCH /v1/me` (`homeRegion`, `homeCurrency`, `units`, `regionSource`); `GET /v1/me` returns them.               |
| `homeCity`            | `{ name, placeRef? }?`                           | For weekend escapes and pre-filled searches.                                                                                                                                                                                                     |
| `homeCurrency`        | string                                           | ISO 4217. Set at onboarding and editable.                                                                                                                                                                                                        |
| `units`               | `{ distance, temperature, clock, firstWeekday }` | `distance` `km` or `mi`; `temperature` `c` or `f`; `clock` `12h` or `24h`; `firstWeekday` 0 to 6. Set at onboarding and editable.                                                                                                                |
| `language`            | string                                           | BCP 47. Drives translated copy.                                                                                                                                                                                                                  |
| `formatLocale`        | string                                           | BCP 47 locale for number and date formats (`en-IN`, `de-DE`). Defaults to the device's.                                                                                                                                                          |
| `regionSource`        | `"device"` / `"confirmed"`                       | How `homeRegion` was set. Device guesses are a hint until confirmed.                                                                                                                                                                             |
| `passportCountry`     | string?                                          | ISO 3166-1 alpha-2. Optional. Drives entry-rule to-dos.                                                                                                                                                                                          |
| `travelStyle`         | `{ interests: string[], pace, spend }`           | Chips from onboarding. `interests`: mountains, beaches, foodTrails, roadTrips, cultureHistory, nightlife, wildlife, slowCafes (max 8, unique). `pace`: slow, balanced, packed. `spend`: shoestring, comfort, splurge. Saved with `PATCH /v1/me`. |
| `paymentHandles`      | `{ type, value }[]`                              | Settle-up handles (`upi`, `venmo`, `paypal`, `wise`, `bank`). Copied to each `members` document by the API.                                                                                                                                      |
| `notifications`       | map of booleans                                  | Push preferences per kind.                                                                                                                                                                                                                       |
| `onboardedAt`         | instant?                                         |                                                                                                                                                                                                                                                  |
| `deletionRequestedAt` | instant?                                         | Set by the API. Deletion needs signal.                                                                                                                                                                                                           |

The phone number is not stored in Firestore. It lives only with the sign-in provider (see [Decisions to confirm](#decisions-to-confirm)). `homeRegion`, `homeCurrency` and `units` are the user-held half of the four locale sources; the trip holds settlement currency and time zone, the destination holds local currency and language, and the app store holds prices.

Subcollections: `devices/{deviceId}` holds `{ platform, pushToken, lastSeenAt }`, owner-only. `stamps/{stampId}` holds `{ tripId, destinationCode, kind, earnedAt }`, written by the server when a trip ends.

### `phoneIndex/{phoneHash}` and `sessions/{sessionId}`

Server-only, written by phone sign-in (VOY-105). No client can read or write them.

`phoneIndex/{phoneHash}` is `{ uid }`. The ID is a keyed HMAC-SHA256 of the E.164 number, so the number itself is never stored and the hash cannot be reversed by trying every number without the server secret. It exists so a returning person finds their account.

`sessions/{sessionId}` is `{ userId, refreshHash, expiresAt, revokedAt? }`, one per signed-in device. `refreshHash` is the SHA-256 of the refresh secret; the secret is never stored. A refresh token is `{sessionId}.{secret}`. Each refresh swaps the hash. Presenting a token whose hash is no longer current revokes the session. Sessions last 60 days from the last refresh.

### `entitlements/{uid}`

Server-owned. The person can read their own. Never trust this document for enforcement in the app; the API asks the entitlements service (VOY-51) on every limited action.

| Field            | Type                                                 | Notes                                                    |
| ---------------- | ---------------------------------------------------- | -------------------------------------------------------- |
| `plus`           | `{ active, productId, store, expiresAt, willRenew }` | Mirrors RevenueCat (D4). In dev everything is unlimited. |
| `trial`          | `{ startedAt, endsAt }?`                             | The 7-day trial.                                         |
| `usage`          | `{ periodStart, askDrafts }`                         | Monthly counters (Ask drafts). Reset by the server.      |
| `limitOverrides` | map?                                                 | Support and dev overrides. Absent means plan defaults.   |

### `trips/{tripId}`

| Field                  | Type                                                           | Notes                                                                                                                                                           |
| ---------------------- | -------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `title`                | string                                                         |                                                                                                                                                                 |
| `ownerUid`             | string                                                         | Mirrors the member with role `owner`.                                                                                                                           |
| `memberUids`           | string[]                                                       | Server-kept mirror for "my trips". Not used for access.                                                                                                         |
| `destinations`         | `{ name, countryCode, placeRef?, timeZone, localCurrency? }[]` | Bounded (say 10). Local currency and language come from the destination, not the person.                                                                        |
| `destinationCountries` | string[]                                                       | ISO alpha-2. Whether the trip is domestic or abroad is worked out per viewer from this and `homeRegion`; it is not stored.                                      |
| `dates`                | `{ start, end }` or `{ flexible: { days, month? } }`           | Calendar dates. A flexible trip is saved as a draft.                                                                                                            |
| `timeZone`             | string                                                         | IANA zone used to display the trip.                                                                                                                             |
| `settlementCurrency`   | string                                                         | ISO 4217. One per trip. Default is the organiser's `homeCurrency`; the owner can change it until the first expense, and then only with a server-run conversion. |
| `budget`               | `{ scope: "perPerson" / "group", amount: Money }?`             | Optional. Without it the screen shows spend only.                                                                                                               |
| `status`               | `"draft"` / `"planned"` / `"active"` / `"past"`                | `past` trips are the ones unlimited on the free plan.                                                                                                           |
| `plan`                 | `{ unlocked, source: "plus" / "pass" / null, byUid?, until? }` | **Server.** Whole-trip features unlock for everyone if any member has Plus or the trip has an active pass. The app only reflects this.                          |
| `usage`                | `{ bestOrderRuns, forwardedBookings, files, photos }`          | **Server.** Counters kept in the same transaction as the create that they count.                                                                                |
| `endedAt`              | instant?                                                       | Set by the server when the trip ends. Triggers stamps.                                                                                                          |

Write: owner edits `title`, `dates`, `destinations`, `budget`; editors edit `title`, `dates`, `destinations`; `plan`, `usage`, `memberUids`, `ownerUid`, `settlementCurrency` (after the first expense), `status` transitions and `endedAt` are **server**. Creating a trip goes through the API because the upcoming-trips limit applies.

### `trips/{tripId}/members/{uid}`

| Field            | Type                                | Notes                                                                |
| ---------------- | ----------------------------------- | -------------------------------------------------------------------- |
| `uid`            | string                              | Same as the document ID.                                             |
| `role`           | `"owner"` / `"editor"` / `"viewer"` |                                                                      |
| `status`         | `"active"` / `"left"` / `"removed"` | Kept after leaving so old expenses still show a name.                |
| `displayName`    | string                              | Snapshot copy, so crew never read another person's `users` document. |
| `photoUrl`       | string?                             |                                                                      |
| `paymentHandles` | `{ type, value }[]`                 | Snapshot so settle-up can open a payment app both people use.        |
| `joinedAt`       | instant                             |                                                                      |
| `invitedVia`     | `{ inviteId }?`                     |                                                                      |

Write: **server only** (join, role change, removal, leave). Read: any active member.

### `invites/{inviteId}`

Server-owned. `inviteId` is a hash of the secret in the link, so the link can be looked up without a query and the secret is never stored. Read and written by the API only.

| Field       | Type                    | Notes          |
| ----------- | ----------------------- | -------------- |
| `tripId`    | string                  |                |
| `role`      | `"editor"` / `"viewer"` | Never `owner`. |
| `createdBy` | string                  |                |
| `expiresAt` | instant                 |                |
| `maxUses`   | integer?                |                |
| `useCount`  | integer                 |                |
| `revokedAt` | instant?                |                |

### `places/{placeId}`

The shared catalogue (one pilot city, Hanoi, until P1). Read by any signed-in user. Written only by the places pipeline (VOY-43).

| Field                 | Type                                      | Notes                                                                                                             |
| --------------------- | ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `name`                | `{ [lang]: string }`                      | Names by BCP 47 language.                                                                                         |
| `location`            | `{ lat, lng }`                            |                                                                                                                   |
| `address`             | map?                                      |                                                                                                                   |
| `categories`          | string[]                                  | Open taxonomy ID plus our grouping for "top 5 per category".                                                      |
| `countryCode`         | string                                    |                                                                                                                   |
| `openingHours`        | map?                                      | Local times with the place's `timeZone`.                                                                          |
| `timeZone`            | string                                    |                                                                                                                   |
| `photos`              | `{ url, attribution, source, licence }[]` | Attribution is shown wherever the photo is.                                                                       |
| `provenance`          | `{ [field]: Provenance }`                 | Source and licence for **every** field above (Foursquare Open Source Places, OSM, Wikidata).                      |
| `googlePlaceId`       | string?                                   | The only Google data kept. Details are fetched live for display and are not stored past the allowed cache window. |
| `wikidataId`, `osmId` | string?                                   | For joining sources.                                                                                              |

### `trips/{tripId}/days/{date}`

One per calendar date of the trip. Holds what is worked out per day, so the app can show it offline. Document ID is the date, `YYYY-MM-DD`.

| Field             | Type                                                                            | Notes                                                                                                                                                             |
| ----------------- | ------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `title`           | string?                                                                         | "Old Quarter day".                                                                                                                                                |
| `travel`          | `{ legs: { fromStopId, toStopId, minutes, approx }[], computedAt, inputHash }?` | Travel times from the directions service. `approx` marks a straight-line estimate. Recomputed (by the server) when `inputHash` no longer matches the day's stops. |
| `lastOptimisedAt` | instant?                                                                        | When "Best order" last ran.                                                                                                                                       |

Write: `title` by editors; `travel` and `lastOptimisedAt` **server**. "Best order" is a limit (once per trip on free), so it runs through the API.

### `trips/{tripId}/stops/{stopId}`

| Field         | Type     | Notes                                                                                     |
| ------------- | -------- | ----------------------------------------------------------------------------------------- |
| `date`        | string   | Calendar date of the day it sits on.                                                      |
| `order`       | string   | Fractional index. Two people inserting at once never clash. Sort by `(date, order)`.      |
| `place`       | PlaceRef |                                                                                           |
| `startLocal`  | string?  | `HH:mm`, local to `timeZone`.                                                             |
| `durationMin` | integer? |                                                                                           |
| `timeZone`    | string   | Defaults to the trip's.                                                                   |
| `note`        | string?  | Plain-text mirror of the collaborative note (Yjs, decided in D3, see [sync.md](sync.md)). |
| `bookingId`   | string?  | When the stop is a booking.                                                               |
| `deletedAt`   | instant? | Tombstone for Restore.                                                                    |

Write: editors and owners. Concurrent edits to different fields both survive.

### `trips/{tripId}/saved/{savedId}`

A place a member saved to the trip before it has a day. These feed crew-ranked recommendations.

| Field     | Type     | Notes                          |
| --------- | -------- | ------------------------------ |
| `place`   | PlaceRef |                                |
| `note`    | string?  |                                |
| `savedBy` | string   | One save per person per place. |

Write: any member for their own saves.

### `trips/{tripId}/bookings/{bookingId}`

| Field                          | Type                                                                   | Notes                                                                                                 |
| ------------------------------ | ---------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `kind`                         | `"flight"` / `"stay"` / `"train"` / `"bus"` / `"activity"` / `"other"` |                                                                                                       |
| `title`                        | string                                                                 |                                                                                                       |
| `provider`                     | string?                                                                | Airline, hotel, operator.                                                                             |
| `startsAt`, `endsAt`           | instant?                                                               | UTC.                                                                                                  |
| `startTimeZone`, `endTimeZone` | string?                                                                | IANA, so a flight shows local times at both ends.                                                     |
| `from`, `to`                   | `{ code?, name }?`                                                     | Airport, station or address labels. Home airport comes from the booking itself.                       |
| `reference`                    | `{ cipher: bytes, keyVersion }?`                                       | Booking reference encrypted with Cloud KMS. **Server** writes and decrypts. Never logged.             |
| `price`                        | Money?                                                                 |                                                                                                       |
| `source`                       | `"manual"` / `"forward"` / `"upload"` / `"gmail"`                      |                                                                                                       |
| `dedupeHash`                   | string?                                                                | Hash of (provider, booking reference, tripId). Two people forwarding the same email give one booking. |
| `fileIds`                      | string[]                                                               | Tickets and confirmations in `files`.                                                                 |
| `extraction`                   | `{ confidence: { [field]: number }, needsReview: boolean }?`           | From the import pipeline; low-confidence fields are flagged for the review screen.                    |

Write: **server** for imported bookings and for any create past the free limit (5 forwarded bookings); editors may edit fields of an existing booking. Offline plan and tickets are never gated.

### `trips/{tripId}/files/{fileId}`

An attached document (ticket, confirmation).

| Field         | Type    | Notes                                                                  |
| ------------- | ------- | ---------------------------------------------------------------------- |
| `name`        | string  |                                                                        |
| `contentType` | string  |                                                                        |
| `sizeBytes`   | integer |                                                                        |
| `storagePath` | string  | Path in the private media bucket. Access is by short-lived signed URL. |
| `bookingId`   | string? |                                                                        |
| `uploadedBy`  | string  |                                                                        |

Write: **server** (it issues the upload URL, which is where the 3-file free limit is checked). Offline tickets are cached on the device.

### `trips/{tripId}/polls/{pollId}` and `votes/{uid}`

| Field      | Type                                                                          | Notes                                 |
| ---------- | ----------------------------------------------------------------------------- | ------------------------------------- |
| `question` | string                                                                        |                                       |
| `options`  | `{ id, label, place?: PlaceRef }[]`                                           | Bounded (say 10).                     |
| `status`   | `"open"` / `"closed"`                                                         |                                       |
| `closesAt` | instant?                                                                      |                                       |
| `closedAt` | instant?                                                                      |                                       |
| `result`   | `{ optionId, counts: { [optionId]: integer }, decidedBy: "vote" / "owner" }?` | **Server.** Set when the poll closes. |

Vote document (`votes/{uid}`, ID is the voter, so voting twice updates): `{ optionIds: string[], votedAt, arrivedAfterClose?: boolean }`. Members write only their own vote while the poll is open. The server alone closes a poll, using the votes it has; later votes are kept and flagged `arrivedAfterClose`.

Write poll: editors and owners. Close and `result`: **server**.

### `trips/{tripId}/tasks/{taskId}`

| Field              | Type                                                      | Notes                                                                                                                                  |
| ------------------ | --------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `title`            | string                                                    |                                                                                                                                        |
| `kind`             | `"custom"` / `"visa"` / `"forex"` / `"sim"` / `"adapter"` | Worked-out to-dos for international trips come from passport country, destination and home plug type. They are not stored on the user. |
| `assigneeUid`      | string?                                                   |                                                                                                                                        |
| `done`             | boolean                                                   | Last writer wins.                                                                                                                      |
| `doneBy`, `doneAt` | string?, instant?                                         |                                                                                                                                        |

Write: editors; any member may tick a task assigned to them.

### `trips/{tripId}/comments/{commentId}`

Append-only. Comments are never edited in place (an edit is a new comment with `replaces`), so nothing conflicts.

| Field       | Type                                                           | Notes                    |
| ----------- | -------------------------------------------------------------- | ------------------------ |
| `target`    | `{ type: "stop" / "saved" / "booking" / "poll" / "trip", id }` |                          |
| `authorUid` | string                                                         |                          |
| `body`      | string                                                         | Free text. Never logged. |
| `reactions` | `{ [uid]: string }`                                            | One reaction per person. |
| `replaces`  | string?                                                        |                          |

Write: any member creates their own comments. Any member may set or clear their own key in `reactions`, and nothing else on someone else's comment.

### `trips/{tripId}/meetPoints/{meetPointId}`

Always free.

| Field   | Type     | Notes         |
| ------- | -------- | ------------- |
| `name`  | string   |               |
| `place` | PlaceRef |               |
| `at`    | instant? | When to meet. |

Write: editors and owners.

### `trips/{tripId}/presence/{uid}`

Opt-in, timed, this trip only. Throwaway: stale is fine, wrong-but-old dots grey out.

| Field       | Type                      | Notes                                                                               |
| ----------- | ------------------------- | ----------------------------------------------------------------------------------- |
| `location`  | `{ lat, lng, accuracyM }` |                                                                                     |
| `updatedAt` | instant                   |                                                                                     |
| `endsAt`    | instant                   | Sharing ends automatically. A Firestore TTL policy deletes the document after this. |

Write: only that person, only while `endsAt` is in the future. Free plan limits sharing to one hour, so `endsAt` is set by the API when sharing starts; the Plus all-day cap is checked there too. Exact locations never go to logs or analytics.

### `trips/{tripId}/expenses/{expenseId}`

The ID is a UUID made on the phone, so retries are safe. Balances are never stored; they are recomputed from the list.

| Field                 | Type                                                                                                                           | Notes                                                                                                                                                                                     |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `description`         | string                                                                                                                         | Free text. Never logged.                                                                                                                                                                  |
| `category`            | string?                                                                                                                        |                                                                                                                                                                                           |
| `paidByUid`           | string                                                                                                                         |                                                                                                                                                                                           |
| `date`                | string                                                                                                                         | Calendar date in the trip's zone.                                                                                                                                                         |
| `original`            | Money                                                                                                                          | What was paid, in the currency it was paid in.                                                                                                                                            |
| `rate`                | Rate?                                                                                                                          | Original to settlement currency. Absent when `original.currency` equals the settlement currency. Saved with the expense and not changed later by a rate refresh.                          |
| `settlement`          | Money                                                                                                                          | `original` converted at `rate`, always in the trip's `settlementCurrency`. This is what balances use.                                                                                     |
| `split`               | `{ method: "equal" / "custom" / "items", shares: { uid, amountMinor }[], items?: { label, amount: Money, uids: string[] }[] }` | Shares are in the settlement currency and sum **exactly** to `settlement.amountMinor`. Leftover minor units go to payers in a fixed, documented order (largest remainder, ties by `uid`). |
| `source`              | `"typed"` / `"receipt"` / `"booking"`                                                                                          |                                                                                                                                                                                           |
| `bookingId`, `fileId` | string?                                                                                                                        | Imported booking or receipt image.                                                                                                                                                        |
| `deletedAt`           | instant?                                                                                                                       | Tombstone. Deleted expenses leave balances at once.                                                                                                                                       |

Write: editors and owners create and edit. A second currency on a trip is a Plus feature (whole-trip), so an expense whose `original.currency` differs from the settlement currency is created through the API, which checks the entitlements service. Rates come from the rate feed through the API; the app never invents one. Free-plan trips use one currency, so those expenses can be written directly. Editing a settled period is blocked while a `paid` settlement exists for it.

### `trips/{tripId}/settlements/{settlementId}`

Server-owned (consistency over availability). The ID is the idempotency key, so "marked as paid" cannot be applied twice.

| Field              | Type                   | Notes                                                            |
| ------------------ | ---------------------- | ---------------------------------------------------------------- |
| `fromUid`, `toUid` | string                 |                                                                  |
| `amount`           | Money                  | Always in the settlement currency.                               |
| `status`           | `"pending"` / `"paid"` |                                                                  |
| `method`           | string?                | The payment app used, or `"manual"`. Voyagami never moves money. |
| `markedBy`         | string                 |                                                                  |
| `paidAt`           | instant?               |                                                                  |

Settle-up is always free and works from any plan.

### `trips/{tripId}/journal/{entryId}`

One entry per person per day. The ID is `{date}_{uid}`.

| Field        | Type                       | Notes                                                                                                         |
| ------------ | -------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `date`       | string                     |                                                                                                               |
| `authorUid`  | string                     |                                                                                                               |
| `visibility` | `"crew"` / `"private"`     | Private entries are readable by the author only. Not even owners.                                             |
| `text`       | string                     | Plain-text mirror for search and export. The collaborative form (Yjs) is added in D3, see [sync.md](sync.md). |
| `mood`       | string?                    | One-tap enum from the journal screen.                                                                         |
| `mediaIds`   | string[]                   | Bounded.                                                                                                      |
| `voice`      | `{ mediaId, transcript }?` | Voice notes are Plus.                                                                                         |

Write: the author only.

### `trips/{tripId}/media/{mediaId}`

| Field             | Type                                | Notes                                                                                                          |
| ----------------- | ----------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `kind`            | `"photo"` / `"voice"` / `"receipt"` |                                                                                                                |
| `uploaderUid`     | string                              |                                                                                                                |
| `storagePath`     | string                              | Private bucket; short-lived signed URLs.                                                                       |
| `contentType`     | string                              |                                                                                                                |
| `sizeBytes`       | integer                             |                                                                                                                |
| `width`, `height` | integer?                            |                                                                                                                |
| `takenAt`         | instant?                            |                                                                                                                |
| `location`        | `{ lat, lng }?`                     | **Present only if the person chose to keep EXIF location.** Otherwise the file is stripped and this is absent. |
| `exifStripped`    | boolean                             |                                                                                                                |
| `thumbnail`       | `{ storagePath, status }`           | Made by a worker.                                                                                              |

Write: **server** creates it when it issues the upload URL (the 30-photo free limit is checked there). The uploader can delete their own.

### `trips/{tripId}/passes/{passId}`

A Trip Pass bought for this trip. **Server** only, from the RevenueCat webhook (D4).

| Field               | Type                              | Notes                                                |
| ------------------- | --------------------------------- | ---------------------------------------------------- |
| `buyerUid`          | string                            | One person can buy for the crew.                     |
| `tier`              | `"weekend"` / `"week"` / `"long"` |                                                      |
| `productId`         | string                            | Price and currency come from the store, not from us. |
| `startsAt`, `until` | instant                           | `until` is 30 days after the trip ends.              |

### `trips/{tripId}/activity/{activityId}`

Append-only feed. It records overwritten values so Restore is one tap.

| Field      | Type           | Notes                                                                         |
| ---------- | -------------- | ----------------------------------------------------------------------------- |
| `actorUid` | string         |                                                                               |
| `kind`     | string         | For example `stop.moved`, `expense.deleted`.                                  |
| `target`   | `{ type, id }` |                                                                               |
| `before`   | map?           | Previous values of the changed fields. Avoid free text where a label will do. |

Write: members append rows where `actorUid` is their own ID; nothing is updated or deleted by clients. Old rows may be pruned by the server.

## Indexes

Defined in code in `firestore.indexes.json` (VOY-97). Firestore builds single-field indexes by itself, so only queries that combine fields need an entry here.

| Collection                | Fields                                                 | Used for                                                                                                                             |
| ------------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------ |
| `trips`                   | `memberUids` (array-contains), `status`, `dates.start` | Trips list split into upcoming and past. Trips with flexible dates have no `dates.start` and do not appear in a query ordered by it. |
| `trips/{tripId}/stops`    | `date`, `order`                                        | Day view in order                                                                                                                    |
| `trips/{tripId}/expenses` | `date` (desc), `createdAt` (desc)                      | Expense list, newest first                                                                                                           |
| `trips/{tripId}/comments` | `target.id`, `createdAt`                               | Comments on one place                                                                                                                |
| `trips/{tripId}/journal`  | `date`, `authorUid`                                    | A day's entries                                                                                                                      |

Also in the file: collection-group indexes on `members.uid` and `members.status` (admin and deletion jobs look people up across trips), and a TTL policy on `trips/{tripId}/presence` field `endsAt`. Bookings (`startsAt`), media (`takenAt`) and the activity feed (`createdAt`) are ordered by one field, so they need nothing extra.

## Offline sync

Local-first: the phone's copy is read first and writes land on the phone first. Detail lives in [sync.md](sync.md) (VOY-62 to VOY-64). The table says only what is cached and what a write needs.

| Collection                                    | On the phone                                                   | Offline write                       | Conflict rule                                                                        |
| --------------------------------------------- | -------------------------------------------------------------- | ----------------------------------- | ------------------------------------------------------------------------------------ |
| `users/{uid}`                                 | Yes                                                            | Yes                                 | Last writer wins per field                                                           |
| `entitlements/{uid}`                          | Read cache                                                     | No                                  | Server                                                                               |
| `trips`, `days`                               | Yes                                                            | Edit (`title`, dates, etc.)         | Last writer wins per field                                                           |
| `members`, `invites`, `passes`, `settlements` | `members`, `passes`, `settlements` read cache; never `invites` | No ("needs signal")                 | Server                                                                               |
| `stops`, `saved`                              | Yes                                                            | Yes                                 | Last writer wins per field; fractional index for order                               |
| `bookings`, `files`                           | Yes, and ticket files downloaded                               | Edit, not create past a limit       | Last writer wins per field. **Offline plan and tickets always work, on every plan.** |
| `polls`, `votes`                              | Yes                                                            | Vote ("pending"); not close         | One vote per person; the server closes                                               |
| `tasks`, `comments`                           | Yes                                                            | Yes                                 | Ticks last writer wins; comments are append-only                                     |
| `meetPoints`                                  | Yes                                                            | Yes                                 | Last writer wins per field                                                           |
| `presence`                                    | No                                                             | No                                  | Throwaway                                                                            |
| `expenses`                                    | Yes                                                            | Yes (same-currency, and edits)      | Last writer wins per field; UUID makes retries safe. Balances recomputed             |
| `journal`                                     | Yes                                                            | Yes                                 | Collaborative text merges (decided in D3)                                            |
| `media`                                       | Yes (thumbnails), originals on demand                          | Queue upload, then server registers | Server                                                                               |
| `activity`                                    | Recent rows                                                    | Append                              | Append-only                                                                          |
| `places`                                      | Only places the trip refers to                                 | No                                  | Server                                                                               |

## Moving to Cloud SQL

Chosen so the P3 move is a mapping, not a redesign:

- Every document under `trips` is a table with a `trip_id` column and a primary key of `(trip_id, id)`; subcollections become tables, and bounded arrays become child tables.
- IDs are strings (UUID v7), not Firestore auto-IDs, and are made on the client where offline matters.
- `Money` is two columns, `amount_minor bigint` and `currency char(3)`.
- No field depends on Firestore-only behaviour (no references, no document-ID ordering, no map keys used as data). The two maps keyed by user ID (`reactions`, vote documents) become child tables.
- Server-kept mirrors (`memberUids`, `usage`, `plan`) are read models. In SQL they are queries or views.
- Rules map to the sync rules per trip plus checks in the API, as the Tech Stack board describes.

## Decisions to confirm

These were not stated in the tickets or boards. The doc takes the first option; change it before VOY-96 and VOY-97 build on it.

1. **Phone number not stored in Firestore.** The Tech Stack board lists phone on `users`. The number is not on the user document, where crew could see it. VOY-105 stores only a keyed hash in the server-only `phoneIndex`, so a returning person finds their account. Invites still match by link or contact, not by number.
2. **Viewer can vote, comment, react, tick their own tasks and write their own journal.** The Plans Spec does not define viewer rights beyond "viewer".
3. **Members can write AP content directly to Firestore** (stops, notes, comments, tasks, expenses on a single currency), while limit-gated creates go through the API. The alternative is to route every write through the API, which is simpler to secure but breaks offline writes.
4. **Trip `memberUids` mirror** for the "my trips" query, kept by the API in the same transaction as the member change. The alternative is a collection-group query on `members`, which costs an extra read per trip.
5. **Presence in Firestore with a TTL** instead of Redis, per the Free-first table on the Tech Stack board.
