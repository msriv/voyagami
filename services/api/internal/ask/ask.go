// Package ask builds a day-by-day draft from catalogue places with a language
// model. The model only picks from the places it is given, and everything it
// returns is checked against that list before anyone sees it.
package ask

import (
	"context"
	"strings"
	"unicode/utf8"
)

// Limits on what goes in and what comes out.
const (
	MaxPrompt   = 1000 // runes, matches the API
	MaxDays     = 14
	MaxPerDay   = 8
	MaxNoteRune = 140
)

// Place is a catalogue place the model may pick.
type Place struct {
	ID    string
	Name  string
	Group string
	Tags  []string
}

// Input is one request. Prompt is the person's words and is treated as data.
type Input struct {
	Prompt string
	Lang   string
	Places []Place
}

// Item is one place on one day.
type Item struct {
	PlaceID string
	Note    string
}

// Day is one day of the draft, numbered from 1.
type Day struct {
	Day   int
	Items []Item
}

// Usage is what the call cost, for logs. It never holds text.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Drafter makes a draft.
type Drafter interface {
	Draft(ctx context.Context, in Input) ([]Day, Usage, error)
}

// Clean keeps only what the person may see: places that were offered, each at
// most once, with short notes and sane day numbers. Days come back in order.
func Clean(offered []Place, raw []Day) []Day {
	known := make(map[string]bool, len(offered))
	for _, p := range offered {
		known[p.ID] = true
	}
	byDay := map[int][]Item{}
	seen := map[string]bool{}
	for _, d := range raw {
		if d.Day < 1 || d.Day > MaxDays {
			continue
		}
		for _, it := range d.Items {
			if !known[it.PlaceID] || seen[it.PlaceID] || len(byDay[d.Day]) == MaxPerDay {
				continue
			}
			seen[it.PlaceID] = true
			byDay[d.Day] = append(byDay[d.Day], Item{PlaceID: it.PlaceID, Note: shorten(it.Note)})
		}
	}
	out := []Day{}
	for n := 1; n <= MaxDays; n++ {
		if items := byDay[n]; len(items) > 0 {
			out = append(out, Day{Day: n, Items: items})
		}
	}
	return out
}

func shorten(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= MaxNoteRune {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:MaxNoteRune-1])) + "…"
}
