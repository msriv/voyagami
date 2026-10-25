package model

import (
	"regexp"
	"strings"
	"time"
)

// Limits on stop fields.
const (
	maxNoteLen     = 2000
	maxPlaceName   = 200
	maxDurationMin = 24 * 60
)

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// PlaceRef points at a place: a catalogue place by ID, or a pin the person dropped.
type PlaceRef struct {
	PlaceID string // set for a catalogue place
	Name    string // set for a custom place
	Lat     float64
	Lng     float64
}

// Custom reports whether this is a dropped pin rather than a catalogue place.
func (p PlaceRef) Custom() bool { return p.PlaceID == "" }

func (p PlaceRef) validate(pr *problems) {
	if !p.Custom() {
		return
	}
	if n := strings.TrimSpace(p.Name); n == "" || len(n) > maxPlaceName {
		pr.addf("place.name is required and at most %d characters", maxPlaceName)
	}
	if p.Lat < -90 || p.Lat > 90 || p.Lng < -180 || p.Lng > 180 {
		pr.addf("place.lat and place.lng must be real coordinates")
	}
}

// Stop is one place on a trip's day, in order.
type Stop struct {
	ID          string
	TripID      string
	Date        string // YYYY-MM-DD
	Order       string // fractional index; sort by (Date, Order)
	Place       PlaceRef
	StartLocal  string // HH:mm, optional
	DurationMin int    // 0 means not set
	TimeZone    string // IANA; defaults to the trip's
	Note        string
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate checks a Stop before it is stored.
func (s Stop) Validate() error {
	var p problems
	if s.ID == "" || s.TripID == "" {
		p.addf("id and tripId are required")
	}
	if !isDate(s.Date) {
		p.addf("date must be YYYY-MM-DD")
	}
	if s.Order == "" {
		p.addf("order is required")
	}
	s.Place.validate(&p)
	validateStopFields(&p, &s.StartLocal, &s.DurationMin, &s.Note)
	if !isTimeZone(s.TimeZone) {
		p.addf("timeZone must be an IANA zone")
	}
	return p.err()
}

// StopPatch changes some fields of a Stop. Nil means leave alone. An empty
// StartLocal or Note, or a zero DurationMin, clears the field.
type StopPatch struct {
	Date        *string
	Order       *string
	StartLocal  *string
	DurationMin *int
	Note        *string
	Deleted     bool // sets the tombstone
}

// Validate checks a patch for the fields it sets.
func (s StopPatch) Validate() error {
	var p problems
	if s.Date != nil && !isDate(*s.Date) {
		p.addf("date must be YYYY-MM-DD")
	}
	if s.Order != nil && *s.Order == "" {
		p.addf("order cannot be empty")
	}
	validateStopFields(&p, s.StartLocal, s.DurationMin, s.Note)
	return p.err()
}

func validateStopFields(p *problems, start *string, dur *int, note *string) {
	if start != nil && *start != "" && !hhmm.MatchString(*start) {
		p.addf("startLocal must be HH:mm")
	}
	if dur != nil && (*dur < 0 || *dur > maxDurationMin) {
		p.addf("durationMin must be between 1 and %d", maxDurationMin)
	}
	if note != nil && len(*note) > maxNoteLen {
		p.addf("note must be at most %d characters", maxNoteLen)
	}
}
