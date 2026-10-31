# Push notifications

## Device tokens

The app registers each install with `PUT /v1/me/devices/{deviceId}` (`platform` ios or android, `pushToken`) after sign-in and whenever the token changes, and removes it with `DELETE` on sign-out. One document per device at `users/{uid}/devices/{deviceId}`; calling again replaces the token. The server reads them with the Admin SDK; clients never read them.

## Sending

`notify.Service.SendToUser(uid, message)` sends to all of a person's devices through a `Notifier`:

- `notify.Log` (dev): sends nothing. Logs the platform, the last four characters of the token and the message `kind`. Never the title, body or data, which can hold names and places.
- `notify.FCM`: Firebase Cloud Messaging HTTP v1, using the API's own service identity (no key). iOS goes through FCM to APNs. The identity needs `roles/firebasecloudmessaging.admin` (granted in `infra/envs/dev/main.tf`).

A token the push service no longer knows (`UNREGISTERED`) deletes that device. One failing device does not stop the others.

Message `data` carries a `kind` and ids for the deep link, never free text typed by people. Real APNs credentials (an APNs key uploaded to the Firebase project) arrive in D4 and P2; until then iOS devices are accepted but pushes to them fail at the push service.

The crew notification tickets (VOY-157 and VOY-158) are what call `SendToUser`.

## Crew notifications

When someone joins a trip, adds a place, starts a poll, adds an expense or adds a meet point, the API sends a push to the other active members. The code is in `internal/notify/crew.go`; handlers report events through `WithCrewEvents`.

- Title is the trip title. Body is the actor's name and what they did, for example "Asha added a place". No free text from people goes in a push.
- Data carries `kind`, `tripId` and `route`. The route opens the matching trip screen (crew, itinerary, polls, money, live).
- Settings: the person's `notifications` map. `all: false` mutes everything; a kind set to false mutes that kind.
- Quiet window: at most one push per kind, trip, actor and person each minute, so a burst of edits is one buzz.
- Dev logs pushes instead of sending them. Other environments send through FCM.

Not done yet: quiet hours (users have no timezone field) and a Pub/Sub queue. The quiet window is in memory per instance.

## Notification settings

`GET /v1/me` returns `notifications` and `PATCH /v1/me` replaces it. Keys are `all` and the five event kinds; a missing key means on; unknown keys answer 400. The app has a Notifications screen under Profile with one switch per kind.

Not done: asking for the push permission at a useful moment and registering the device token. The onboarding screen asks for permission already, but registering a token needs Firebase config files from Mihir. Quiet hours and per-trip mute are not built (quiet hours need a user timezone).
