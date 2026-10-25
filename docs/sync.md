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

## Live changes (change feed)

`GET /v1/trips/{id}/events` is a server-sent event stream for one trip. It tells the app what changed so it can refetch, and carries no content.

- Each `change` event is `{kind, id, op}`. Kinds are trip, members, stops, tasks, expenses, settlements, bookings, polls, meetPoints and comments. Journal entries and live locations are not on the feed because they are private to one person.
- The API opens one Firestore listener per kind, so any Cloud Run instance can serve the stream. The first snapshot of each listener is the current state and is not sent.
- Nothing is replayed. The stream ends after about four minutes. The app reconnects with `Last-Event-ID`, and the first event is then `resync`: refetch everything for the trip, then apply `change` events as they arrive.
- Any active member can listen, viewers included.
