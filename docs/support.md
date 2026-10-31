# Support

The Help screen sends a message with `POST /v1/support/messages` (VOY-79).

- Body: `category` (billing, account, bug, other), `text` (1 to 4000 characters) and optional `diagnostics` (app version, OS, OS version, device model, locale, a request id). Diagnostics carry no phone number, location or trip content.
- Each message is stored in the `supportMessages` collection with the sender's id. The text is free text, so it is never written to logs; the server logs nothing about it.
- One person may send 5 messages, then 1 every 12 minutes. The limit is per server instance (`shortcut` in `internal/server/support.go`).
- The collection has no client access (see `firestore.rules`). Read it in the Firebase console until a help desk is chosen.
- Not built here: the help centre articles and the public status page. They need content and a host, so they stay on the ticket.
