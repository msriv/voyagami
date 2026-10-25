package model

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// ValidationError means a value is not acceptable. It lists every problem.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "invalid: " + strings.Join(e.Problems, "; ")
}

// IsValidation reports whether err is, or wraps, a ValidationError.
func IsValidation(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

type problems []string

func (p *problems) addf(format string, args ...any) {
	*p = append(*p, fmt.Sprintf(format, args...))
}

func (p problems) err() error {
	if len(p) == 0 {
		return nil
	}
	return &ValidationError{Problems: p}
}

// These checks are about shape only. Whether a code is a real ISO 4217
// currency, with its decimals, belongs to the locale module (VOY-58).

func isUpper(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

func isDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func isMonth(s string) bool {
	_, err := time.Parse("2006-01", s)
	return err == nil
}

func isTimeZone(s string) bool {
	if s == "" {
		return false
	}
	_, err := time.LoadLocation(s)
	return err == nil
}

func (m Money) validate(p *problems, field string) {
	if !isUpper(m.Currency, 3) {
		p.addf("%s currency must be a 3-letter ISO 4217 code", field)
	}
}

// Choices the travel style screen offers. The API checks them so a bad client cannot store junk.
var (
	TravelInterests = []string{"mountains", "beaches", "foodTrails", "roadTrips", "cultureHistory", "nightlife", "wildlife", "slowCafes"}
	TravelPaces     = []string{"slow", "balanced", "packed"}
	TravelSpends    = []string{"shoestring", "comfort", "splurge"}
)

func (t TravelStyle) validate(p *problems) {
	seen := map[string]bool{}
	for _, i := range t.Interests {
		if !slices.Contains(TravelInterests, i) {
			p.addf("travelStyle.interests has an unknown choice")
		} else if seen[i] {
			p.addf("travelStyle.interests has a repeat")
		}
		seen[i] = true
	}
	if len(t.Interests) > len(TravelInterests) {
		p.addf("travelStyle.interests has too many choices")
	}
	if t.Pace != "" && !slices.Contains(TravelPaces, t.Pace) {
		p.addf("travelStyle.pace is not a known choice")
	}
	if t.Spend != "" && !slices.Contains(TravelSpends, t.Spend) {
		p.addf("travelStyle.spend is not a known choice")
	}
}

func (u Units) validate(p *problems) {
	if u.Distance != "km" && u.Distance != "mi" {
		p.addf("units.distance must be km or mi")
	}
	if u.Temperature != "c" && u.Temperature != "f" {
		p.addf("units.temperature must be c or f")
	}
	if u.Clock != "12h" && u.Clock != "24h" {
		p.addf("units.clock must be 12h or 24h")
	}
	if u.FirstWeekday < 0 || u.FirstWeekday > 6 {
		p.addf("units.firstWeekday must be 0 to 6")
	}
}

// Validate checks a User before it is stored.
func (u User) Validate() error {
	var p problems
	if u.ID == "" {
		p.addf("id is required")
	}
	if !isUpper(u.HomeRegion, 2) {
		p.addf("homeRegion must be a 2-letter ISO 3166-1 code")
	}
	if !isUpper(u.HomeCurrency, 3) {
		p.addf("homeCurrency must be a 3-letter ISO 4217 code")
	}
	if u.PassportCountry != "" && !isUpper(u.PassportCountry, 2) {
		p.addf("passportCountry must be a 2-letter ISO 3166-1 code")
	}
	if u.Language == "" {
		p.addf("language is required")
	}
	if u.RegionSource != "device" && u.RegionSource != "confirmed" {
		p.addf("regionSource must be device or confirmed")
	}
	u.Units.validate(&p)
	return p.err()
}

// Validate checks a patch for the fields it sets.
func (u UserPatch) Validate() error {
	var p problems
	if u.HomeRegion != nil && !isUpper(*u.HomeRegion, 2) {
		p.addf("homeRegion must be a 2-letter ISO 3166-1 code")
	}
	if u.HomeCurrency != nil && !isUpper(*u.HomeCurrency, 3) {
		p.addf("homeCurrency must be a 3-letter ISO 4217 code")
	}
	if u.PassportCountry != nil && *u.PassportCountry != "" && !isUpper(*u.PassportCountry, 2) {
		p.addf("passportCountry must be a 2-letter ISO 3166-1 code")
	}
	if u.Language != nil && *u.Language == "" {
		p.addf("language cannot be empty")
	}
	if u.RegionSource != nil && *u.RegionSource != "device" && *u.RegionSource != "confirmed" {
		p.addf("regionSource must be device or confirmed")
	}
	if u.Units != nil {
		u.Units.validate(&p)
	}
	if u.TravelStyle != nil {
		u.TravelStyle.validate(&p)
	}
	if u.DisplayName != nil && (strings.TrimSpace(*u.DisplayName) == "" || len([]rune(*u.DisplayName)) > 80) {
		p.addf("displayName must be 1 to 80 characters")
	}
	return p.err()
}

func (d TripDates) validate(p *problems) {
	fixed := d.Start != "" || d.End != ""
	switch {
	case fixed && d.Flexible != nil:
		p.addf("dates must be fixed or flexible, not both")
	case fixed:
		if !isDate(d.Start) || !isDate(d.End) {
			p.addf("dates.start and dates.end must be YYYY-MM-DD")
		} else if d.End < d.Start {
			p.addf("dates.end cannot be before dates.start")
		}
	case d.Flexible != nil:
		if d.Flexible.Days < 1 {
			p.addf("dates.flexible.days must be at least 1")
		}
		if d.Flexible.Month != "" && !isMonth(d.Flexible.Month) {
			p.addf("dates.flexible.month must be YYYY-MM")
		}
	default:
		p.addf("dates are required")
	}
}

func validateDestinations(p *problems, ds []Destination) {
	if len(ds) == 0 {
		p.addf("at least one destination is required")
	}
	if len(ds) > 10 {
		p.addf("at most 10 destinations")
	}
	for i, d := range ds {
		if d.Name == "" {
			p.addf("destinations[%d].name is required", i)
		}
		if !isUpper(d.CountryCode, 2) {
			p.addf("destinations[%d].countryCode must be a 2-letter ISO 3166-1 code", i)
		}
		if !isTimeZone(d.TimeZone) {
			p.addf("destinations[%d].timeZone must be an IANA zone", i)
		}
		if d.LocalCurrency != "" && !isUpper(d.LocalCurrency, 3) {
			p.addf("destinations[%d].localCurrency must be a 3-letter ISO 4217 code", i)
		}
	}
}

func (b Budget) validate(p *problems) {
	if b.Scope != "perPerson" && b.Scope != "group" {
		p.addf("budget.scope must be perPerson or group")
	}
	if b.Amount.AmountMinor < 0 {
		p.addf("budget.amount cannot be negative")
	}
	b.Amount.validate(p, "budget.amount")
}

func validTripStatus(s TripStatus) bool {
	switch s {
	case TripDraft, TripPlanned, TripActive, TripPast:
		return true
	}
	return false
}

// Validate checks a Trip before it is stored.
func (t Trip) Validate() error {
	var p problems
	if t.ID == "" {
		p.addf("id is required")
	}
	if strings.TrimSpace(t.Title) == "" {
		p.addf("title is required")
	}
	validateDestinations(&p, t.Destinations)
	t.Dates.validate(&p)
	if !isTimeZone(t.TimeZone) {
		p.addf("timeZone must be an IANA zone")
	}
	if !isUpper(t.SettlementCurrency, 3) {
		p.addf("settlementCurrency must be a 3-letter ISO 4217 code")
	}
	if t.Budget != nil {
		t.Budget.validate(&p)
	}
	if !validTripStatus(t.Status) {
		p.addf("status must be draft, planned, active or past")
	}
	return p.err()
}

// Validate checks a patch for the fields it sets.
func (t TripPatch) Validate() error {
	var p problems
	if t.Title != nil && strings.TrimSpace(*t.Title) == "" {
		p.addf("title cannot be empty")
	}
	if t.Destinations != nil {
		validateDestinations(&p, *t.Destinations)
	}
	if t.Dates != nil {
		t.Dates.validate(&p)
	}
	if t.TimeZone != nil && !isTimeZone(*t.TimeZone) {
		p.addf("timeZone must be an IANA zone")
	}
	if t.Budget != nil {
		t.Budget.validate(&p)
	}
	if t.Status != nil && !validTripStatus(*t.Status) {
		p.addf("status must be draft, planned, active or past")
	}
	return p.err()
}

// Validate checks a Member before it is stored.
func (m Member) Validate() error {
	var p problems
	if m.UID == "" {
		p.addf("uid is required")
	}
	switch m.Role {
	case RoleOwner, RoleEditor, RoleViewer:
	default:
		p.addf("role must be owner, editor or viewer")
	}
	switch m.Status {
	case MemberActive, MemberLeft, MemberRemoved:
	default:
		p.addf("status must be active, left or removed")
	}
	if m.DisplayName == "" {
		p.addf("displayName is required")
	}
	return p.err()
}

// MaxTaskTitle is the longest task title, in characters.
const MaxTaskTitle = 120

func validTaskTitle(p *problems, title string) {
	t := strings.TrimSpace(title)
	if t == "" {
		p.addf("title is required")
	} else if utf8.RuneCountInString(t) > MaxTaskTitle {
		p.addf("title must be at most %d characters", MaxTaskTitle)
	}
}

// Validate checks a Task before it is stored.
func (t Task) Validate() error {
	var p problems
	if t.ID == "" {
		p.addf("id is required")
	}
	validTaskTitle(&p, t.Title)
	return p.err()
}

// Validate checks a patch for the fields it sets.
func (t TaskPatch) Validate() error {
	var p problems
	if t.Title != nil {
		validTaskTitle(&p, *t.Title)
	}
	return p.err()
}
