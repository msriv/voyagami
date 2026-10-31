# Product analytics

Built for VOY-75. The app sends events to `POST /v1/events`; the server checks them and stores the good ones in the `analyticsEvents` Firestore collection. No third-party analytics SDK is used. Counting funnels is done by querying that collection (BigQuery export of Firestore is the next step when the weekly review needs a dashboard; that needs Mihir to turn on the export, so it stays on the ticket).

## Rules

- Events are kept only for people whose `analytics` consent is `granted` (`GET /v1/me/consents`). Without it, the server drops the whole batch and says so in the reply.
- Event names come from a fixed list (below). An unknown name is dropped.
- A batch has 1 to 50 events. An event older than 7 days (phones may hold events while offline) or more than 5 minutes in the future is dropped.
- At most 10 properties per event. Property names are `camelCase` words. A value is an identifier (letters, digits, `_ . : -`, up to 64 characters), a number or a boolean. Anything else is dropped, so free text (it has spaces) cannot get in.
- A property whose name contains `phone`, `email`, `name`, `address`, `lat`, `lng`, `lon`, `location`, `text`, `note`, `message` or `token` is dropped. This is blunt on purpose; pick another word.
- Each stored event has the person's id, so funnels can count people. It has no phone number, location or content.

## Events

| Name                   | When                                | Useful properties                       |
| ---------------------- | ----------------------------------- | --------------------------------------- |
| `app_installed`        | First launch                        |                                         |
| `sign_in`              | Sign-in succeeds                    | `isNewUser`                             |
| `onboarding_completed` | Onboarding is done                  |                                         |
| `trip_created`         | A trip is made                      | `destinations` (count), `hasDates`      |
| `invite_sent`          | An invite link is shared            | `channel`                               |
| `crew_joined`          | Someone joins a trip                |                                         |
| `place_added`          | A stop is added                     | `source` (search, picks, pin)           |
| `expense_added`        | An expense is saved                 | `split` (equal, exact, percent, shares) |
| `journal_entry_added`  | A journal entry is saved            |                                         |
| `ask_draft_created`    | An Ask draft is made                |                                         |
| `trial_started`        | A trial starts                      |                                         |
| `subscription_started` | Plus is bought                      | `product` (store product id)            |
| `trip_pass_bought`     | A Trip Pass is bought               |                                         |
| `paywall_shown`        | The paywall appears                 | `moment` (see below)                    |
| `paywall_dismissed`    | The paywall is closed               | `moment`                                |
| `paywall_converted`    | A purchase is made from the paywall | `moment`, `product`                     |
| `screen_viewed`        | A screen opens                      | `route`                                 |

`moment` is the `feature` of the limit that opened the paywall (for example `bestOrder`, `topPicks`, `journalPhotos`), or `trial_end` or `plans`.

## Funnels

install (`app_installed`) to sign-in (`sign_in`) to first trip (`trip_created`) to invite sent (`invite_sent`) to crew joined (`crew_joined`) to first expense (`expense_added`) to trial start (`trial_started`) to paid (`subscription_started` or `trip_pass_bought`).

Paywall: for each `moment`, `paywall_shown` against `paywall_dismissed` and `paywall_converted`.

## Not done yet

- The weekly dashboard (needs the BigQuery export and a viewer).
- Removing a person's events when their account is deleted. They carry the id but no contact details; add it to the purge when the events move to BigQuery.
