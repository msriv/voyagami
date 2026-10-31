# Flight status

`GET /v1/trips/{tripId}/bookings/{bookingId}/flight-status` asks a status provider about a flight booking.

## What is built

- The route, the `flightstatus.Provider` interface and the code that finds an airline code and number (such as "AI 302") in the booking title.
- Answers 404 for a booking that is not a flight or has no flight number in its title, and 503 when no provider is set up or the provider fails.

## What is not

No provider is wired in, so every call answers 503. Choosing one is a paid-service decision for Mihir (VOY-68 stays open). To finish it: write a `Provider` for the chosen service, add its key as a Secret Manager secret, and pass it to `server.WithFlightStatus` in `cmd/api/signin.go`.

## Known limits

- Bookings have no flight number field, so the number is read from the title. Add a field when the booking form gets one.
- The provider is called on every request. Add a short cache before the app polls.
