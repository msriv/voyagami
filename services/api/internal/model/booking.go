package model

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// BookingKinds are the kinds of booking a trip can hold.
var BookingKinds = []string{"flight", "stay", "train", "bus", "activity", "other"}

// BookingPlace is where a booking starts or ends: an airport, station or address label.
type BookingPlace struct {
	Code string
	Name string
}

// Booking is a flight, stay or ticket on a trip. Times are UTC instants; the
// zones say how to show them locally.
type Booking struct {
	ID            string
	Kind          string
	Title         string
	Provider      string
	StartsAt      time.Time
	EndsAt        time.Time
	StartTimeZone string
	EndTimeZone   string
	From, To      *BookingPlace
	Price         *Money
	CreatedBy     string
	CreatedAt     time.Time
}

func (b *BookingPlace) validate(p *problems, field string) {
	if b == nil {
		return
	}
	if strings.TrimSpace(b.Name) == "" || utf8.RuneCountInString(b.Name) > 120 {
		p.addf("%s name is required and at most 120 characters", field)
	}
	if utf8.RuneCountInString(b.Code) > 10 {
		p.addf("%s code must be at most 10 characters", field)
	}
}

// Validate checks a Booking before it is stored.
func (b Booking) Validate() error {
	var p problems
	if b.ID == "" {
		p.addf("id is required")
	}
	if !slices.Contains(BookingKinds, b.Kind) {
		p.addf("kind must be one of %s", strings.Join(BookingKinds, ", "))
	}
	if t := strings.TrimSpace(b.Title); t == "" || utf8.RuneCountInString(t) > 120 {
		p.addf("title is required and at most 120 characters")
	}
	if utf8.RuneCountInString(b.Provider) > 120 {
		p.addf("provider must be at most 120 characters")
	}
	if !b.StartsAt.IsZero() && !b.EndsAt.IsZero() && b.EndsAt.Before(b.StartsAt) {
		p.addf("endsAt must not be before startsAt")
	}
	for _, z := range []string{b.StartTimeZone, b.EndTimeZone} {
		if z != "" && !isTimeZone(z) {
			p.addf("time zones must be IANA names such as Asia/Tokyo")
		}
	}
	b.From.validate(&p, "from")
	b.To.validate(&p, "to")
	if b.Price != nil {
		b.Price.validate(&p, "price")
	}
	return p.err()
}
