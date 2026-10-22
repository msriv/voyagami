// Package model holds the plain types the API stores and returns. They follow
// docs/data-model.md and know nothing about Firestore.
package model

import (
	"time"

	"github.com/google/uuid"
)

// NewID returns a time-ordered UUID v7 string, used for documents the caller
// creates. Making IDs on the caller's side keeps retried writes idempotent.
func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// Money is an amount in integer minor units plus an ISO 4217 currency code.
// Never use a float for money.
type Money struct {
	AmountMinor int64
	Currency    string
}

// Role is a person's role on one trip.
type Role string

// The roles a member can have on a trip.
const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// MemberStatus says whether a member still has access. Members who leave or
// are removed keep their document so old expenses still show a name.
type MemberStatus string

// The states of a trip membership.
const (
	MemberActive  MemberStatus = "active"
	MemberLeft    MemberStatus = "left"
	MemberRemoved MemberStatus = "removed"
)

// TripStatus is where a trip is in its life.
type TripStatus string

// The stages of a trip.
const (
	TripDraft   TripStatus = "draft"
	TripPlanned TripStatus = "planned"
	TripActive  TripStatus = "active"
	TripPast    TripStatus = "past"
)

// PaymentHandle is how a person gets paid back, such as a UPI ID or a PayPal
// address. Type is a code the settle-up screen understands; no payment app is
// assumed.
type PaymentHandle struct {
	Type  string
	Value string
}

// Units are the viewer's display preferences.
type Units struct {
	Distance     string // "km" or "mi"
	Temperature  string // "c" or "f"
	Clock        string // "12h" or "24h"
	FirstWeekday int    // 0 (Sunday) to 6
}

// HomeCity is the person's home city, used for weekend escapes and search.
type HomeCity struct {
	Name string
}

// TravelStyle holds the chips picked during onboarding.
type TravelStyle struct {
	Interests []string
	Pace      string
	Spend     string
}

// User is the person's own profile. No phone number is stored here.
type User struct {
	ID              string
	DisplayName     string
	PhotoURL        string
	HomeRegion      string // ISO 3166-1 alpha-2
	HomeCity        *HomeCity
	HomeCurrency    string // ISO 4217
	Units           Units
	Language        string // BCP 47
	FormatLocale    string // BCP 47
	RegionSource    string // "device" or "confirmed"
	PassportCountry string // ISO 3166-1 alpha-2, optional
	TravelStyle     TravelStyle
	PaymentHandles  []PaymentHandle
	Notifications   map[string]bool
	OnboardedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// UserPatch changes some fields of a User. Nil means leave alone.
type UserPatch struct {
	DisplayName     *string
	PhotoURL        *string
	HomeRegion      *string
	HomeCity        *HomeCity
	HomeCurrency    *string
	Units           *Units
	Language        *string
	FormatLocale    *string
	RegionSource    *string
	PassportCountry *string
	TravelStyle     *TravelStyle
	PaymentHandles  *[]PaymentHandle
	Notifications   *map[string]bool
	OnboardedAt     *time.Time
}

// Destination is one place a trip goes. Local currency and language come from
// the destination, not from the traveller.
type Destination struct {
	Name          string
	CountryCode   string // ISO 3166-1 alpha-2
	TimeZone      string // IANA
	LocalCurrency string // ISO 4217, optional
}

// FlexibleDates is a trip with no fixed days yet, such as "9 days in Nov".
type FlexibleDates struct {
	Days  int
	Month string // "YYYY-MM", optional
}

// TripDates holds either fixed calendar dates or a flexible length, never both.
type TripDates struct {
	Start    string // YYYY-MM-DD
	End      string // YYYY-MM-DD
	Flexible *FlexibleDates
}

// Budget is the optional trip budget.
type Budget struct {
	Scope  string // "perPerson" or "group"
	Amount Money
}

// TripPlan says whether whole-trip features are unlocked. Only the
// entitlements service writes it.
type TripPlan struct {
	Unlocked bool
	Source   string // "plus", "pass" or ""
}

// TripUsage counts what the free-plan limits count. Only the API changes it.
type TripUsage struct {
	BestOrderRuns     int
	ForwardedBookings int
	Files             int
	Photos            int
}

// Trip is a trip and the summary data every member can read.
type Trip struct {
	ID                   string
	Title                string
	OwnerUID             string
	MemberUIDs           []string
	Destinations         []Destination
	DestinationCountries []string
	Dates                TripDates
	TimeZone             string // IANA
	SettlementCurrency   string // ISO 4217; one per trip
	Budget               *Budget
	Status               TripStatus
	Plan                 TripPlan
	Usage                TripUsage
	EndedAt              *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TripPatch changes some fields of a Trip. Nil means leave alone.
type TripPatch struct {
	Title        *string
	Destinations *[]Destination
	Dates        *TripDates
	TimeZone     *string
	Budget       *Budget
	Status       *TripStatus
}

// Member is one person's place on one trip.
type Member struct {
	UID            string
	Role           Role
	Status         MemberStatus
	DisplayName    string
	PhotoURL       string
	PaymentHandles []PaymentHandle
	InvitedVia     string // invite ID, if any
	JoinedAt       time.Time
}

// Session is one signed-in device. The refresh token itself is never stored,
// only its hash.
type Session struct {
	ID          string
	UserID      string
	RefreshHash string // hex SHA-256 of the refresh secret
	ExpiresAt   time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
