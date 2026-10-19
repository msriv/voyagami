# Data model

> Stub. Filled in by VOY-10 and by each ticket that adds or changes stored data. See the Data Map board (`design/boards/`, listed in [BOARD_INDEX](../design/BOARD_INDEX.md)).

## Store

Firestore at launch, in the `voyagami-dev` project during the D phases. A later move to Cloud SQL and PowerSync is planned (P3); keep shapes easy to map to tables.

## Rules every collection follows

- **Money** is an integer amount in minor units plus an ISO-4217 currency code. Never floats. Decimals per currency come from ISO data (JPY and VND have 0).
- **Expenses** store the original amount, the original currency and the rate used to convert to the trip's settlement currency.
- **Locale** is never hard-coded. The user holds home region and units, the trip holds settlement currency and timezone, the destination holds local currency and language, and the app store holds prices.
- **Timestamps** are UTC. Display uses the trip's timezone.
- **Places data** keeps its source and licence per field. Google Places data is not stored beyond the allowed cache window.
- **Downgrades** never delete data; over-limit content becomes read-only.

## Collections

<!-- One section per collection: path, fields with types, indexes, who can read and write it, and the ticket that added it. -->

_None yet._
