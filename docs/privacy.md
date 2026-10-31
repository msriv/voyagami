# Privacy: consents, export and deletion

Built for VOY-72. Today only consents exist; export and deletion follow.

## Consents

`GET /v1/me/consents` and `PATCH /v1/me/consents` keep one choice per purpose: location, notifications, photos and analytics. Each is `unset` (never asked), `granted` or `denied`, and can be changed later. A patch changes only the purposes it sends and can never send `unset`.

Stored in `consents/{uid}`. The phone also keeps the choices on the device so the switches work offline. The server's copy is what the server obeys: product events (`POST /v1/events`) are dropped unless `analytics` is `granted`.
