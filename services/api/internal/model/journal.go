package model

import (
	"slices"
	"time"
	"unicode/utf8"
)

// Journal visibility values.
const (
	VisibilityCrew    = "crew"
	VisibilityPrivate = "private"
)

// JournalMoods are the one-tap moods. This list is provisional until the
// journal screen is built; the screen should use whatever this says.
var JournalMoods = []string{"happy", "excited", "calm", "grateful", "tired", "meh"}

// Journal limits.
const (
	MaxJournalText  = 10000
	MaxJournalStops = 20
)

// JournalEntry is one person's note for one day of a trip.
type JournalEntry struct {
	Date       string
	AuthorUID  string
	Visibility string
	Text       string
	Mood       string
	StopIDs    []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ID is the document ID: one entry per person per day.
func (e JournalEntry) ID() string { return e.Date + "_" + e.AuthorUID }

// Validate checks a JournalEntry before it is stored.
func (e JournalEntry) Validate() error {
	var p problems
	if !isDate(e.Date) {
		p.addf("date must be YYYY-MM-DD")
	}
	if e.AuthorUID == "" {
		p.addf("author is required")
	}
	if e.Visibility != VisibilityCrew && e.Visibility != VisibilityPrivate {
		p.addf("visibility must be crew or private")
	}
	if utf8.RuneCountInString(e.Text) > MaxJournalText {
		p.addf("text must be at most %d characters", MaxJournalText)
	}
	if e.Mood != "" && !slices.Contains(JournalMoods, e.Mood) {
		p.addf("mood is not one of the offered choices")
	}
	if len(e.StopIDs) > MaxJournalStops {
		p.addf("at most %d linked stops", MaxJournalStops)
	}
	return p.err()
}
