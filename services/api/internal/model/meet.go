package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Limits on meet points and shared locations.
const (
	MaxMeetName = 100
	// MeetDuration is how long a meet point with a time stays active after it.
	MeetDuration = 3 * time.Hour
	// MeetOpenWindow is how long a meet point with no time stays active after it is made.
	MeetOpenWindow = 24 * time.Hour
	// DefaultShareMinutes is how long location sharing runs when nobody says.
	DefaultShareMinutes = 60
	// MaxShareMinutes is the longest a single share can run (one day).
	MaxShareMinutes = 24 * 60
)

// MeetPoint is a place and time the crew agrees to meet.
type MeetPoint struct {
	ID          string
	Name        string
	Place       PlaceRef
	At          time.Time // zero means no time set
	CreatedBy   string
	CreatedAt   time.Time
	ConfirmedBy []string // members who said they are coming
}

// EndsAt is when the meet is over: a few hours after its time, or a day after
// it was made if it has no time.
func (m MeetPoint) EndsAt() time.Time {
	if !m.At.IsZero() {
		return m.At.Add(MeetDuration)
	}
	return m.CreatedAt.Add(MeetOpenWindow)
}

// Validate checks a meet point before it is stored.
func (m MeetPoint) Validate() error {
	var p problems
	if m.ID == "" || m.CreatedBy == "" {
		p.addf("id and createdBy are required")
	}
	if n := strings.TrimSpace(m.Name); n == "" || utf8.RuneCountInString(n) > MaxMeetName {
		p.addf("name is required and at most %d characters", MaxMeetName)
	}
	if m.Place.PlaceID == "" && m.Place.Name == "" {
		p.addf("place is required")
	}
	m.Place.validate(&p)
	return p.err()
}

// MeetPointPatch changes some fields of a meet point. Nil means leave alone.
type MeetPointPatch struct {
	Name  *string
	Place *PlaceRef
	At    *time.Time
}

// Validate checks a patch for the fields it sets.
func (m MeetPointPatch) Validate() error {
	var p problems
	if m.Name != nil {
		if n := strings.TrimSpace(*m.Name); n == "" || utf8.RuneCountInString(n) > MaxMeetName {
			p.addf("name must be 1 to %d characters", MaxMeetName)
		}
	}
	if m.Place != nil {
		if m.Place.PlaceID == "" && m.Place.Name == "" {
			p.addf("place is required")
		}
		m.Place.validate(&p)
	}
	return p.err()
}

// Presence is where one person is sharing their location. It is opt-in, ends
// by itself at EndsAt, and covers one trip.
type Presence struct {
	UID         string
	Lat, Lng    float64
	AccuracyM   int
	MeetPointID string // set if the share is for one meet and ends with it
	UpdatedAt   time.Time
	EndsAt      time.Time
}

// Validate checks a shared location.
func (s Presence) Validate() error {
	var p problems
	if s.UID == "" {
		p.addf("uid is required")
	}
	if s.Lat < -90 || s.Lat > 90 || s.Lng < -180 || s.Lng > 180 {
		p.addf("lat and lng must be real coordinates")
	}
	if s.AccuracyM < 0 || s.AccuracyM > 100000 {
		p.addf("accuracyM must be between 0 and 100000")
	}
	if !s.EndsAt.After(s.UpdatedAt) {
		p.addf("sharing must end in the future")
	}
	return p.err()
}
