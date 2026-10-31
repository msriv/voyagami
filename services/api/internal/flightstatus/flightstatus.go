// Package flightstatus is the seam for a live flight status provider. No
// provider is chosen yet (it is a paid service), so the API answers 503 until
// one is wired in.
package flightstatus

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// Flight identifies one flight on one day.
type Flight struct {
	Carrier string // IATA airline code, e.g. "AI"
	Number  string // digits only, e.g. "302"
	Date    time.Time
}

// State is where the flight is in its life.
type State string

// The states the API reports.
const (
	Scheduled State = "scheduled"
	Delayed   State = "delayed"
	Cancelled State = "cancelled"
	Departed  State = "departed"
	Landed    State = "landed"
	Unknown   State = "unknown"
)

// Leg is one end of a flight.
type Leg struct {
	Scheduled, Estimated time.Time
	Terminal, Gate       string
}

// Status is a provider's answer.
type Status struct {
	State     State
	Departure Leg
	Arrival   Leg
	UpdatedAt time.Time
}

// Provider looks a flight up. Implementations must not log the flight number
// together with a person.
type Provider interface {
	Status(ctx context.Context, f Flight) (Status, error)
}

var code = regexp.MustCompile(`\b([A-Z][A-Z0-9]|[A-Z0-9][A-Z])\s?(\d{1,4})\b`)

// Parse finds an airline code and flight number such as "AI 302" in a booking
// title. ok is false when there is none.
func Parse(title string, date time.Time) (f Flight, ok bool) {
	m := code.FindStringSubmatch(strings.ToUpper(title))
	if m == nil {
		return Flight{}, false
	}
	return Flight{Carrier: m[1], Number: strings.TrimLeft(m[2], "0"), Date: date}, true
}
