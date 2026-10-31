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
