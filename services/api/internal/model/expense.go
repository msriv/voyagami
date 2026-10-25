package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Rate converts original minor units to settlement minor units:
// settlement = original x Numerator / Denominator, rounded half away from zero.
type Rate struct {
	Numerator   int64
	Denominator int64
}

// Share is one person's part of an expense, in the settlement currency.
type Share struct {
	UID         string
	AmountMinor int64
}

// Expense is one thing somebody paid for on a trip. Balances are never stored;
// they are worked out from the list.
type Expense struct {
	ID          string
	Description string
	Category    string
	PaidByUID   string
	Date        string
	Original    Money
	Rate        *Rate // nil when Original is already in the settlement currency
	Settlement  Money
	SplitMethod string // equal, exact, percent or shares
	Shares      []Share
	CreatedBy   string
	CreatedAt   time.Time
}

// Longest description and category, in characters.
const (
	MaxExpenseDescription = 120
	MaxExpenseCategory    = 40
)

// Validate checks an Expense before it is stored.
func (e Expense) Validate() error {
	var p problems
	if e.ID == "" {
		p.addf("id is required")
	}
	if d := strings.TrimSpace(e.Description); d == "" {
		p.addf("description is required")
	} else if utf8.RuneCountInString(d) > MaxExpenseDescription {
		p.addf("description must be at most %d characters", MaxExpenseDescription)
	}
	if utf8.RuneCountInString(e.Category) > MaxExpenseCategory {
		p.addf("category must be at most %d characters", MaxExpenseCategory)
	}
	if e.PaidByUID == "" {
		p.addf("paidByUid is required")
	}
	if !isDate(e.Date) {
		p.addf("date must be YYYY-MM-DD")
	}
	e.Original.validate(&p, "original")
	e.Settlement.validate(&p, "settlement")
	if e.Original.AmountMinor == 0 {
		p.addf("original amount must not be zero")
	}
	if (e.Original.Currency == e.Settlement.Currency) != (e.Rate == nil) {
		p.addf("rate is needed when the currency differs from the trip's, and not otherwise")
	}
	if e.Rate != nil && (e.Rate.Numerator < 1 || e.Rate.Denominator < 1) {
		p.addf("rate must be positive")
	}
	var sum int64
	for _, s := range e.Shares {
		sum += s.AmountMinor
	}
	if len(e.Shares) == 0 || sum != e.Settlement.AmountMinor {
		p.addf("shares must add up to the settlement amount")
	}
	return p.err()
}

// Settlement is a payment one member says they made to another. Voyagami
// never moves money; this only changes balances.
type Settlement struct {
	ID        string
	FromUID   string
	ToUID     string
	Amount    Money // always in the trip's settlement currency
	Method    string
	MarkedBy  string
	CreatedAt time.Time
}

// Validate checks a Settlement before it is stored.
func (s Settlement) Validate() error {
	var p problems
	if s.ID == "" {
		p.addf("id is required")
	}
	if s.FromUID == "" || s.ToUID == "" || s.FromUID == s.ToUID {
		p.addf("a payment needs two different people")
	}
	if s.Amount.AmountMinor < 1 {
		p.addf("amount must be more than zero")
	}
	s.Amount.validate(&p, "amount")
	if utf8.RuneCountInString(s.Method) > 20 {
		p.addf("method must be at most 20 characters")
	}
	return p.err()
}
