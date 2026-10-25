# Dev logs

A dev-only sink so a developer can read what a phone reported (crashes, errors) without the phone attached. It exists only when `APP_ENV=dev`; staging and prod do not serve it. It is outside the OpenAPI contract on purpose.

## Send a report

`POST /v1/dev/logs`, no sign-in, up to 32 KB, about 20 reports in a burst then one every 2 seconds per caller. Answers 204.

```json
{
  "deviceId": "string, required",
  "appVersion": "string",
  "level": "debug | info | warn | error | fatal",
  "message": "string, required",
  "stack": "string",
  "route": "string",
  "breadcrumbs": ["short strings, newest last"]
}
```

Long fields are trimmed (message 2000 characters, stack 8000, 30 breadcrumbs of 200). The app must not put phone numbers, exact locations or free text a person typed into a report.

## Read reports

`GET /v1/dev/logs?deviceId=&since=&limit=` returns `{"entries": [...]}`, newest first. All three parameters are optional. `since` is an RFC 3339 time and `limit` is 1 to 200 (default 50). Anyone who can reach the dev URL can read it, so keep the data free of personal details.

Entries live in the `devLogs` Firestore collection and are never deleted automatically.
