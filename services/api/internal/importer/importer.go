// Package importer turns a pasted confirmation or a PDF into candidate
// bookings. The model call sits behind Parser so tests and dev can swap it.
package importer

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

// Limits on what a caller may send.
const (
	MaxTextRunes = 20000
	MaxPDFBytes  = 4 << 20
	MaxResults   = 10
)

// ErrNothingFound means the document held no booking the parser could read.
var ErrNothingFound = errors.New("importer: no bookings found")

// Input is one document: pasted text or a PDF, never both.
type Input struct {
	Text string
	PDF  []byte
}

// Candidate is one booking the parser found. Confidence runs from 0 to 1.
// The Booking has no ID: the caller creates it only after the person agrees.
type Candidate struct {
	Booking    model.Booking
	Confidence float64
}

// Parser reads a document and returns the bookings in it. The document is
// data: a Parser must never act on instructions found inside it.
type Parser interface {
	Parse(ctx context.Context, in Input) ([]Candidate, error)
}

// Raw is the shape the model is asked to answer in. Times are RFC 3339.
type Raw struct {
	Kind          string  `json:"kind"`
	Title         string  `json:"title"`
	Provider      string  `json:"provider"`
	StartsAt      string  `json:"startsAt"`
	EndsAt        string  `json:"endsAt"`
	StartTimeZone string  `json:"startTimeZone"`
	EndTimeZone   string  `json:"endTimeZone"`
	FromCode      string  `json:"fromCode"`
	FromName      string  `json:"fromName"`
	ToCode        string  `json:"toCode"`
	ToName        string  `json:"toName"`
	AmountMinor   int64   `json:"amountMinor"`
	Currency      string  `json:"currency"`
	Confidence    float64 `json:"confidence"`
}

// Normalize turns the model's answer into candidates. Anything that would not
// pass booking validation is dropped, so a bad answer can never reach a trip.
// Confidence is clamped to 0..1 and at most MaxResults are kept.
func Normalize(raw []Raw) []Candidate {
	var out []Candidate
	for _, r := range raw {
		b := model.Booking{
			ID: "candidate", Kind: strings.ToLower(strings.TrimSpace(r.Kind)),
			Title: strings.TrimSpace(r.Title), Provider: strings.TrimSpace(r.Provider),
			StartTimeZone: strings.TrimSpace(r.StartTimeZone), EndTimeZone: strings.TrimSpace(r.EndTimeZone),
		}
		b.StartsAt = parseTime(r.StartsAt)
		b.EndsAt = parseTime(r.EndsAt)
		b.From = place(r.FromCode, r.FromName)
		b.To = place(r.ToCode, r.ToName)
		if r.AmountMinor > 0 && len(r.Currency) == 3 {
			b.Price = &model.Money{AmountMinor: r.AmountMinor, Currency: strings.ToUpper(r.Currency)}
		}
		if b.Validate() != nil {
			// Try again without the optional parts that models get wrong most.
			b.Price, b.StartTimeZone, b.EndTimeZone = nil, "", ""
			if b.Validate() != nil {
				continue
			}
		}
		b.ID = ""
		out = append(out, Candidate{Booking: b, Confidence: clamp(r.Confidence)})
		if len(out) == MaxResults {
			break
		}
	}
	return out
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func place(code, name string) *model.BookingPlace {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	if name == "" {
		name = code
	}
	if name == "" {
		return nil
	}
	return &model.BookingPlace{Code: code, Name: name}
}

func clamp(f float64) float64 {
	switch {
	case f < 0:
		return 0
	case f > 1:
		return 1
	}
	return f
}
