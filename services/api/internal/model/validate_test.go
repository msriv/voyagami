package model

import (
	"strings"
	"testing"
)

func goodTrip() Trip {
	return Trip{
		ID:                 "t1",
		Title:              "Trip",
		Destinations:       []Destination{{Name: "Hanoi", CountryCode: "VN", TimeZone: "Asia/Ho_Chi_Minh"}},
		Dates:              TripDates{Start: "2026-11-21", End: "2026-11-29"},
		TimeZone:           "Asia/Ho_Chi_Minh",
		SettlementCurrency: "EUR",
		Status:             TripPlanned,
	}
}

func TestTripValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Trip)
		wantErr string
	}{
		{"valid", func(*Trip) {}, ""},
		{"valid flexible", func(tr *Trip) { tr.Dates = TripDates{Flexible: &FlexibleDates{Days: 9, Month: "2026-11"}} }, ""},
		{"no id", func(tr *Trip) { tr.ID = "" }, "id is required"},
		{"blank title", func(tr *Trip) { tr.Title = "  " }, "title is required"},
		{"no destinations", func(tr *Trip) { tr.Destinations = nil }, "at least one destination"},
		{"too many destinations", func(tr *Trip) { tr.Destinations = make([]Destination, 11) }, "at most 10"},
		{"bad country", func(tr *Trip) { tr.Destinations[0].CountryCode = "Vietnam" }, "countryCode"},
		{"bad zone", func(tr *Trip) { tr.Destinations[0].TimeZone = "Nowhere/Land" }, "timeZone"},
		{"end before start", func(tr *Trip) { tr.Dates = TripDates{Start: "2026-11-29", End: "2026-11-21"} }, "cannot be before"},
		{"not a date", func(tr *Trip) { tr.Dates = TripDates{Start: "21/11/2026", End: "29/11/2026"} }, "YYYY-MM-DD"},
		{"fixed and flexible", func(tr *Trip) { tr.Dates.Flexible = &FlexibleDates{Days: 3} }, "not both"},
		{"no dates", func(tr *Trip) { tr.Dates = TripDates{} }, "dates are required"},
		{"zero flexible days", func(tr *Trip) { tr.Dates = TripDates{Flexible: &FlexibleDates{}} }, "at least 1"},
		{"bad flexible month", func(tr *Trip) { tr.Dates = TripDates{Flexible: &FlexibleDates{Days: 2, Month: "Nov"}} }, "YYYY-MM"},
		{"lowercase currency", func(tr *Trip) { tr.SettlementCurrency = "eur" }, "settlementCurrency"},
		{"negative budget", func(tr *Trip) {
			tr.Budget = &Budget{Scope: "group", Amount: Money{AmountMinor: -1, Currency: "EUR"}}
		}, "negative"},
		{"bad budget scope", func(tr *Trip) {
			tr.Budget = &Budget{Scope: "everyone", Amount: Money{AmountMinor: 1, Currency: "EUR"}}
		}, "budget.scope"},
		{"bad status", func(tr *Trip) { tr.Status = "maybe" }, "status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := goodTrip()
			tt.mutate(&tr)
			err := tr.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !IsValidation(err) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want validation error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestMemberValidate(t *testing.T) {
	good := Member{UID: "u", Role: RoleEditor, Status: MemberActive, DisplayName: "A"}
	if err := good.Validate(); err != nil {
		t.Fatalf("good member rejected: %v", err)
	}
	for name, mutate := range map[string]func(*Member){
		"no uid":     func(m *Member) { m.UID = "" },
		"bad role":   func(m *Member) { m.Role = "admin" },
		"bad status": func(m *Member) { m.Status = "away" },
		"no name":    func(m *Member) { m.DisplayName = "" },
	} {
		m := good
		mutate(&m)
		if err := m.Validate(); !IsValidation(err) {
			t.Errorf("%s: error = %v, want validation error", name, err)
		}
	}
}

func TestNewIDIsUniqueAndTimeOrdered(t *testing.T) {
	a, b := NewID(), NewID()
	if a == b || len(a) != 36 {
		t.Fatalf("ids %q and %q", a, b)
	}
	if a[14] != '7' {
		t.Errorf("id %q is not a UUID v7", a)
	}
}

func TestUserPatchValidate(t *testing.T) {
	name := func(s string) *string { return &s }
	ok := UserPatch{DisplayName: name("Asha"), TravelStyle: &TravelStyle{Interests: []string{"beaches", "slowCafes"}, Pace: "slow", Spend: "comfort"}}
	if err := ok.Validate(); err != nil {
		t.Fatalf("good patch rejected: %v", err)
	}
	for label, p := range map[string]UserPatch{
		"blank name":       {DisplayName: name("  ")},
		"long name":        {DisplayName: name(strings.Repeat("a", 81))},
		"unknown interest": {TravelStyle: &TravelStyle{Interests: []string{"skydiving"}}},
		"duplicate":        {TravelStyle: &TravelStyle{Interests: []string{"beaches", "beaches"}}},
		"too many":         {TravelStyle: &TravelStyle{Interests: []string{"mountains", "beaches", "foodTrails", "roadTrips", "cultureHistory", "nightlife", "wildlife", "slowCafes", "x"}}},
		"bad pace":         {TravelStyle: &TravelStyle{Pace: "zoom"}},
		"bad spend":        {TravelStyle: &TravelStyle{Spend: "lavish"}},
	} {
		if err := p.Validate(); !IsValidation(err) {
			t.Errorf("%s: error = %v, want validation error", label, err)
		}
	}
}
