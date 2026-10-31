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
