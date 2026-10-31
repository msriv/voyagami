# Stays: partner hand-off links

The app shows where to stay by sending people to a booking partner. We do not take bookings or payments. Partners come from configuration, so no partner is named in code.

## Configuration

`STAY_PARTNERS` is a JSON list. With nothing set the list is empty and the app shows no stay buttons.

```json
[
  {
    "id": "example",
    "name": "Example Stays",
    "baseUrl": "https://stays.example.com/search",
    "params": {
      "destination": "q",
      "checkIn": "checkin",
      "checkOut": "checkout",
      "guests": "adults"
    },
    "fixed": { "aid": "PLACEHOLDER" }
  }
]
```

- `id` and `name` are required, `id` is unique. `baseUrl` must be https.
- `params` gives the partner's query parameter names. A missing name leaves that piece out.
- `fixed` holds parameters sent on every link, such as the affiliate tracking id. Real IDs come with the P2 partner accounts.
- A bad list stops the API from starting, so a typo shows up at deploy, not in the app.

## Routes

- `GET /v1/trips/{tripId}/stays/links?guests=2` returns one link per partner. It searches the trip's first destination and its dates (YYYY-MM-DD, left out when missing). A trip with no destination answers 400. Members only.
- `POST /v1/trips/{tripId}/stays/clicks` with `{ "partnerId": "..." }` counts a tap. Unknown partners answer 400. Members only.

## Privacy

The link carries the destination, dates, guest count and fixed tracking ids. It carries nothing about the person. A click is stored only as a count per partner and UTC day in `stayClicks/{partnerId}_{day}`, with no person, trip or place.

## Not done

The Stays screen is VOY-187. Prices and availability from partner APIs (with an "as of" time) are in the epic VOY-46 but not built. Dates in other formats than YYYY-MM-DD per partner are not supported yet; add a per-partner format when a partner needs one.
