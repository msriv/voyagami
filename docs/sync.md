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
