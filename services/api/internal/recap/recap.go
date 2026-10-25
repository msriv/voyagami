// Package recap assembles the facts a trip recap shows. It is pure: the caller
// loads the data and decides what the person may see.
package recap

import (
	"sort"
	"time"
	"unicode/utf8"

	"github.com/msriv/voyagami/services/api/internal/bestorder"
	"github.com/msriv/voyagami/services/api/internal/model"
)

// MaxHighlights is how many journal highlights a recap carries.
const MaxHighlights = 5

// highlightRunes is the longest highlight snippet.
const highlightRunes = 140

// Input is everything a recap is built from. Entries must already be limited
// to what the viewer may see (crew entries plus their own private ones).
type Input struct {
	Trip    model.Trip
	Stops   []model.Stop // live stops only
	Entries []model.JournalEntry
	// SpentMinor is the total spent in the trip's settlement currency.
	SpentMinor int64
	// Balances are what each person is owed (positive) or owes (negative); nil
	// when settle-up is off.
	Balances map[string]int64
}

// Highlight is a short piece of a journal entry.
type Highlight struct {
	Date      string
	AuthorUID string
	Text      string
}

// Day is a date and a count of things on it.
type Day struct {
	Date  string
	Count int
}

// Result is the recap's facts.
type Result struct {
	Days           int
	Stops          int
	Crew           int
	DistanceMeters int64
	SpentMinor     int64
	Currency       string
	TargetMinor    *int64
	RemainingMinor *int64
	// SettledUp is nil when it cannot be known.
	SettledUp *bool
	// FavouriteDay is the day with the most journal entries, earliest on a tie.
	FavouriteDay *Day
	Highlights   []Highlight
}

// Build assembles a recap. The same input always gives the same result.
func Build(in Input) Result {
	r := Result{
		Stops: len(in.Stops), Crew: len(in.Trip.MemberUIDs),
		SpentMinor: in.SpentMinor, Currency: in.Trip.SettlementCurrency,
		Days: tripDays(in.Trip, in.Stops),
	}
	r.DistanceMeters = distanceMeters(in.Stops)

	if b := in.Trip.Budget; b != nil && b.Amount.Currency == in.Trip.SettlementCurrency {
		target := b.Amount.AmountMinor
		if b.Scope == "perPerson" {
			target *= int64(r.Crew)
		}
		rem := target - in.SpentMinor
		r.TargetMinor, r.RemainingMinor = &target, &rem
	}
	if in.Balances != nil {
		settled := true
		for _, v := range in.Balances {
			if v != 0 {
				settled = false
				break
			}
		}
		r.SettledUp = &settled
	}

	perDay := map[string]int{}
	var withText []model.JournalEntry
	for _, e := range in.Entries {
		perDay[e.Date]++
		if e.Text != "" {
			withText = append(withText, e)
		}
	}
	if len(perDay) > 0 {
		dates := make([]string, 0, len(perDay))
		for d := range perDay {
			dates = append(dates, d)
		}
		sort.Strings(dates)
		best := dates[0]
		for _, d := range dates {
			if perDay[d] > perDay[best] {
				best = d
			}
		}
		r.FavouriteDay = &Day{Date: best, Count: perDay[best]}
	}
	sort.Slice(withText, func(i, j int) bool {
		if withText[i].Date != withText[j].Date {
			return withText[i].Date < withText[j].Date
		}
		return withText[i].AuthorUID < withText[j].AuthorUID
	})
	for _, e := range withText {
		if len(r.Highlights) == MaxHighlights {
			break
		}
		r.Highlights = append(r.Highlights, Highlight{Date: e.Date, AuthorUID: e.AuthorUID, Text: snippet(e.Text)})
	}
	return r
}

// tripDays counts the days of a dated trip, or the distinct days with stops
// when the trip has no fixed dates.
func tripDays(t model.Trip, stops []model.Stop) int {
	if t.Dates.Start != "" && t.Dates.End != "" {
		s, err1 := time.Parse(time.DateOnly, t.Dates.Start)
		e, err2 := time.Parse(time.DateOnly, t.Dates.End)
		if err1 == nil && err2 == nil && !e.Before(s) {
			return int(e.Sub(s).Hours()/24) + 1
		}
	}
	seen := map[string]bool{}
	for _, s := range stops {
		seen[s.Date] = true
	}
	return len(seen)
}

// distanceMeters adds straight-line legs between consecutive pinned stops on
// each day. Stops from the catalogue have no coordinates here and are skipped.
// shortcut: straight lines, not routes; swap for the directions service (VOY-128).
func distanceMeters(stops []model.Stop) int64 {
	byDay := map[string][]model.Stop{}
	for _, s := range stops {
		if s.Place.Custom() {
			byDay[s.Date] = append(byDay[s.Date], s)
		}
	}
	var km float64
	for _, day := range byDay {
		sort.Slice(day, func(i, j int) bool { return day[i].Order < day[j].Order })
		for i := 1; i < len(day); i++ {
			km += bestorder.DistanceKm(
				bestorder.Point{Lat: day[i-1].Place.Lat, Lng: day[i-1].Place.Lng},
				bestorder.Point{Lat: day[i].Place.Lat, Lng: day[i].Place.Lng})
		}
	}
	return int64(km*1000 + 0.5)
}

func snippet(s string) string {
	if utf8.RuneCountInString(s) <= highlightRunes {
		return s
	}
	r := []rune(s)
	return string(r[:highlightRunes]) + "…"
}
