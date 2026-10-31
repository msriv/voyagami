# Ask drafts

`POST /v1/trips/{tripId}/ask/drafts` turns a request like "slow food trip, two days" into a day-by-day draft. It is a preview: nothing is added to the trip until the person picks items and adds them as stops.

## How it works

1. The server finds the trip's centre (the middle of its stops that have a position) and takes up to 60 catalogue places within 10 km.
2. It sends those places and the person's text to Gemini on Vertex, through the API's own service account.
3. The reply is cleaned (`ask.Clean`): only places we offered are kept, each at most once, with notes cut to 140 characters and days limited to 1 to 14. Names come from the catalogue, never from the model.

Nothing is invented. A place the model made up is dropped.

## Safety

- The person's text goes inside a `<request>` block and the system prompt says it is untrusted data. Even if it gets through, the output filter above bounds what comes back.
- Logs hold request ID, latency and token counts only. No prompt, no draft, no places.

## Limits

- Editor or owner role. Counted per person per month through the entitlements service (feature `askDrafts`): Free 3, Plus fair use. A refused call answers 403 `limit_reached` with the feature before the model is called.
- A draft is counted only when it returns at least one item. A model failure answers 503 and costs nothing.
- `remaining` in the answer is drafts left this month, or -1 for no cap.

## Setup

Uses the same `VERTEX_LOCATION`, `GCP_PROJECT` and `VERTEX_MODEL` as booking import (see `infra/README.md`). Without them the route answers 503.

## Not done

- Trips need at least one stop with a position, because destinations have no coordinates yet.
- Limit check and count are two steps, so two calls at once can slip one extra draft through.
