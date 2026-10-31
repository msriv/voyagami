# Sync

> Stub. Filled in by the D3 offline tickets (VOY-62, VOY-63, VOY-64). See the Offline Sync board in [BOARD_INDEX](../design/BOARD_INDEX.md).

## Goals

- The app is local-first: reads come from the device, and writes land on the device first and sync when online.
- The offline plan and tickets always work offline, on every plan.
- The user can always see sync status (synced, pending, failed) per item.

## To decide and document here

- Local store on the device and how it maps to Firestore collections ([data-model.md](data-model.md)).
- Write queue: ordering, retries, idempotency keys.
- Conflict rules per kind of data (itinerary order, expenses, journal entries, polls).
- How live co-editing (D2) and offline edits meet.
- What changes for the planned Cloud SQL and PowerSync move (P3).

## Local store (VOY-166)

`apps/mobile/src/local/store.ts` defines `LocalStore` (`get`, `put`, `remove`, `clear`, `sizeBytes`). The app uses the SQLite implementation (`expo-sqlite`, one table `docs(key, json, updated_at)`). Tests use the in-memory one. Screens never touch SQLite, so a PowerSync move (P3) swaps only this file.

Keys are the screen's whole payload as the API returned it: `trips`, `stops:{tripId}` (trip and stops), `bookings:{tripId}`, `expenses:{tripId}` (trip, expenses, budget, members), `journal:{tripId}`.

`localFirst(key, fetch, onData)` is how screens read: it shows the device copy straight away, asks the server, then shows and saves the fresh answer. If the fresh read fails and a copy exists, the copy stays on screen. A 404 or 403 drops the copy (the trip is gone or no longer shared). Signing out clears the whole store.

**Migrations.** `PRAGMA user_version` holds the schema version (now 1). The documents are a cache of server data, so a layout change that cannot be migrated in place drops the table and the next read refills it. Nothing is lost that the server does not hold; writes made offline live in the queue (VOY-167), which has its own table and must be migrated, not dropped.

## Live changes (change feed)

`GET /v1/trips/{id}/events` is a server-sent event stream for one trip. It tells the app what changed so it can refetch, and carries no content.

- Each `change` event is `{kind, id, op}`. Kinds are trip, members, stops, tasks, expenses, settlements, bookings, polls, meetPoints and comments. Journal entries and live locations are not on the feed because they are private to one person.
- The API opens one Firestore listener per kind, so any Cloud Run instance can serve the stream. The first snapshot of each listener is the current state and is not sent.
- Nothing is replayed. The stream ends after about four minutes. The app reconnects with `Last-Event-ID`, and the first event is then `resync`: refetch everything for the trip, then apply `change` events as they arrive.
- Any active member can listen, viewers included.

## Concurrent edit rules

These hold for every shared record (stops, tasks, bookings, expenses, polls, meet points). The server applies them; the app never merges.

- **Per field, last write wins.** A patch changes only the fields it names (Firestore field updates). Two people changing different fields of one stop both keep their change. Two people changing the same field: the later write stands, and the earlier one is replaced. Nothing else on the record is touched.
- **Order uses a sort key.** Stops are ordered by a fractional key (`fracindex`). Moving or inserting writes only the moved stop's key, so two people reordering at once do not overwrite each other's moves. If two stops end up with the same key, they sort by id, so every device shows the same order.
- **Deletes win.** A delete marks the record deleted (`deletedAt`). An edit that arrives after the delete is refused with 404; it never brings the record back. To undo, add the item again.
- **Nothing is lost silently.** An edit that is refused answers with an error the app shows. The live feed (above) tells everyone to refetch, so a device that lost a race sees the winning value within seconds.

`TestStopsConcurrentEdits` and `TestStopsConcurrentInserts` in `services/api/internal/server/stops_test.go` prove the first three rules for stops.
