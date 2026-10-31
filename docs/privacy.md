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

## Account deletion

`DELETE /v1/me` schedules deletion 30 days out and answers with the date. The person stays signed in and `DELETE /v1/me/deletion` cancels it; `GET /v1/me/deletion` says whether one is scheduled. Asking again keeps the first date. While a subscription still renews (`subscription.willRenew`) it answers 409 with code `subscription_active`, so the person cancels in the store first. A trial does not block.

A daily job (`POST /v1/jobs/account-deletions`, same secret as the other jobs, see `infra/README.md`) purges accounts past their date, up to 50 a run. Purging is safe to repeat. It deletes:

- the profile, devices, stamps and progress, consent choices, entitlements, sessions and the phone link;
- every trip the person was alone on, with everything in it;
- on shared trips: their journal entries and live location. Their member record stays as someone who left, with no name, photo or payment details, so expenses and comments still make sense. If they owned the trip, the member who joined first becomes the owner.

Expenses they paid, comments, votes and stops they added stay on shared trips; they point to a person who no longer exists, so the app shows "Former member" for an empty name. Product events and support messages from them are not removed yet (they carry the id but no contact details).

The Plus subscription itself lives in the store; deleting the account does not cancel it, which is why the 409 exists.
