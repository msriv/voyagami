# Privacy: consents, export and deletion

Built for VOY-72. Today only consents exist; export and deletion follow.

## Consents

`GET /v1/me/consents` and `PATCH /v1/me/consents` keep one choice per purpose: location, notifications, photos and analytics. Each is `unset` (never asked), `granted` or `denied`, and can be changed later. A patch changes only the purposes it sends and can never send `unset`.

Stored in `consents/{uid}`. The phone also keeps the choices on the device so the switches work offline. The server's copy is what the server obeys: product events (`POST /v1/events`) are dropped unless `analytics` is `granted`.

## Data export

`GET /v1/me/export` returns a ZIP, free on every plan. It holds `profile.json`, `consents.json`, `trips.json`, and for each trip the person is on, under `trips/{tripId}/`: `stops.json`, `expenses.json`, `expenses.csv`, `bookings.json` and `journal.json`. The journal file has only the entries the person wrote. The CSV writes amounts as integer minor units with the ISO currency code, and a text cell that starts with `=`, `+`, `-` or `@` gets a leading apostrophe so a spreadsheet does not run it.

JSON files use the stored field names. Photos are not included yet because there is no photo upload (VOY-161); add them when it lands.

## Export and deletion in the app (VOY-72)

- **Data and privacy** has "Export my data". It downloads the ZIP from `GET /v1/me/export` and opens the share sheet. A build made before `expo-file-system` and `expo-sharing` were added shows a plain message instead of failing, so the dev client should be rebuilt once.
- "Delete my account" asks first, then calls `DELETE /v1/me`. The screen shows the deletion date and a "Keep my account" button that calls `DELETE /v1/me/deletion`. If a subscription is running the server answers `subscription_active` and the screen asks the person to cancel it in the store first.
- Each consent switch is also sent to `PATCH /v1/me/consents`. On sign-in the choices already on the server are loaded and win over this phone's.
