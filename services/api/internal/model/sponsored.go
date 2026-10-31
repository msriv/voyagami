package model

import "time"

// HideSponsoredFor is how long a hidden sponsored card stays away.
const HideSponsoredFor = 30 * 24 * time.Hour

// SponsoredCard is one direct deal from the inventory table. Targeting is by
// destination country and list group only; nothing about the person.
type SponsoredCard struct {
	ID       string
	Sponsor  string
	Title    string
	Body     string
	ImageURL string
	URL      string
	// Countries are the destination countries (ISO 3166-1 alpha-2) the card runs
	// in. Empty means every country.
	Countries []string
	// Groups are the place groups of the lists it may sit in. Empty means any list.
	Groups   []PlaceGroup
	Priority int // higher wins when several match
	StartsAt time.Time
	EndsAt   time.Time // zero means no end
}

// LiveAt reports whether now is inside the card's dates.
func (c SponsoredCard) LiveAt(now time.Time) bool {
	return !now.Before(c.StartsAt) && (c.EndsAt.IsZero() || now.Before(c.EndsAt))
}

// Runs reports whether the card is live at now, in one of the countries, in a
// list of the group (empty group means the trip-wide list).
func (c SponsoredCard) Runs(now time.Time, countries []string, group PlaceGroup) bool {
	if !c.LiveAt(now) {
		return false
	}
	if len(c.Countries) > 0 && !anyIn(c.Countries, countries) {
		return false
	}
	if len(c.Groups) == 0 || group == "" {
		return true
	}
	for _, g := range c.Groups {
		if g == group {
			return true
		}
	}
	return false
}

func anyIn(want, have []string) bool {
	for _, w := range want {
		for _, h := range have {
			if w == h {
				return true
			}
		}
	}
	return false
}

// SponsoredEvent is what a person did with a card.
type SponsoredEvent string

// The two things counted.
const (
	SponsoredImpression SponsoredEvent = "impression"
	SponsoredClick      SponsoredEvent = "click"
)
