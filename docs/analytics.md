# Product analytics

We keep our own small event pipeline: the app sends batches to `POST /v1/events` and the server stores counts. No third-party analytics tool is used, so nothing leaves our own project and no paid account is needed. The weekly review reads the stored counts.

## Rules

- The app sends nothing until the person turns on **Help improve the app** in Privacy. Turning it off stops sending at once and drops anything still waiting.
- Events carry a name, a time and a few short properties. Never a phone number, a place, a trip name or any free text.
- Properties are short fixed words (like a paywall moment), numbers or booleans.
- The server drops any name not on the list below.

## Events

| Name                | When                                                  | Properties                |
| ------------------- | ----------------------------------------------------- | ------------------------- |
| `app_open`          | The app opens, or a person signs in with analytics on | none                      |
| `trip_created`      | A trip is created none                                |
| `invite_sent`       | An invite is shared                                   | none                      |
| `crew_joined`       | A person joins a trip by invite                       | none                      |
| `expense_added`     | An expense is saved none                              |
| `trial_started`     | A free trial begins (server)                          | none                      |
| `paid`              | A purchase completes (server)                         | `plan` (`plus` or `pass`) |
| `paywall_shown`     | The paywall opens                                     | `moment`                  |
| `paywall_dismissed` | "Not now" or close                                    | `moment`                  |
| `paywall_converted` | "Try Plus" is tapped                                  | `moment`                  |

## Funnel

install (first `app_open`), sign-in, first trip, invite sent, crew joined, first expense, trial start, paid. Each step is the first time its event appears for a person. `trial_started` and `paid` are sent by the server from store notifications, not by the app.

## Weekly review

Count events per week by name and divide each funnel step by the one before it. Paywall conversion is `paywall_converted` over `paywall_shown`, per `moment`.

## Server checks

`POST /v1/events` (VOY-75) checks each batch before storing it in the `analyticsEvents` collection.

- Kept only for people whose `analytics` consent is `granted` (`GET /v1/me/consents`). Without it the whole batch is dropped and the reply says so.
- A batch has 1 to 50 events. An event older than 7 days or more than 5 minutes in the future is dropped.
- At most 10 properties per event, with `camelCase` names. A value is an identifier (letters, digits, `_ . : -`, up to 64 characters), a number or a boolean. Free text has spaces, so it cannot get in.
- A property whose name contains `phone`, `email`, `name`, `address`, `lat`, `lng`, `lon`, `location`, `text`, `note`, `message` or `token` is dropped.
- Each stored event has the person's id so funnels can count people, and nothing else about them.

## Weekly review

`GET /v1/jobs/analytics-report?days=28` returns counts as JSON: per event the number of events and of different people, the funnel (people who did each step in the period; steps are not forced into order), and the paywall outcome per moment. It takes the same bearer secret as the scheduled jobs and has no app screen:

```sh
curl -H "Authorization: Bearer $JOBS_SECRET" "$API_URL/v1/jobs/analytics-report?days=7"
```

It scans stored events (capped at 100,000), which is fine while Mihir is the only user. It holds no phone number, place or text, only counts.

## Not done yet

- A chart dashboard. Turn on the BigQuery export of Firestore (needs Mihir) when the JSON stops being enough.
- Deleting an account does not yet remove its events. They carry the id but no contact details. Add it to the purge when events move to BigQuery.
