package firestorerepo

import (
	"sort"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

// The *Doc types are what is stored. They are separate from the model types so
// storage names and Firestore tags stay out of the rest of the API.

type unitsDoc struct {
	Distance     string `firestore:"distance"`
	Temperature  string `firestore:"temperature"`
	Clock        string `firestore:"clock"`
	FirstWeekday int    `firestore:"firstWeekday"`
}

type homeCityDoc struct {
	Name string `firestore:"name"`
}

type travelStyleDoc struct {
	Interests []string `firestore:"interests"`
	Pace      string   `firestore:"pace"`
	Spend     string   `firestore:"spend"`
}

type paymentHandleDoc struct {
	Type  string `firestore:"type"`
	Value string `firestore:"value"`
}

type userDoc struct {
	DisplayName     string             `firestore:"displayName"`
	PhotoURL        string             `firestore:"photoUrl,omitempty"`
	HomeRegion      string             `firestore:"homeRegion"`
	HomeCity        *homeCityDoc       `firestore:"homeCity,omitempty"`
	HomeCurrency    string             `firestore:"homeCurrency"`
	Units           unitsDoc           `firestore:"units"`
	Language        string             `firestore:"language"`
	FormatLocale    string             `firestore:"formatLocale,omitempty"`
	RegionSource    string             `firestore:"regionSource"`
	PassportCountry string             `firestore:"passportCountry,omitempty"`
	TravelStyle     travelStyleDoc     `firestore:"travelStyle"`
	PaymentHandles  []paymentHandleDoc `firestore:"paymentHandles"`
	Notifications   map[string]bool    `firestore:"notifications,omitempty"`
	OnboardedAt     *time.Time         `firestore:"onboardedAt,omitempty"`
	CreatedAt       time.Time          `firestore:"createdAt,serverTimestamp"`
	UpdatedAt       time.Time          `firestore:"updatedAt,serverTimestamp"`
}

func handlesToDoc(in []model.PaymentHandle) []paymentHandleDoc {
	out := make([]paymentHandleDoc, len(in))
	for i, h := range in {
		out[i] = paymentHandleDoc{Type: h.Type, Value: h.Value}
	}
	return out
}

func handlesFromDoc(in []paymentHandleDoc) []model.PaymentHandle {
	if len(in) == 0 {
		return nil
	}
	out := make([]model.PaymentHandle, len(in))
	for i, h := range in {
		out[i] = model.PaymentHandle{Type: h.Type, Value: h.Value}
	}
	return out
}

func unitsToDoc(u model.Units) unitsDoc {
	return unitsDoc{Distance: u.Distance, Temperature: u.Temperature, Clock: u.Clock, FirstWeekday: u.FirstWeekday}
}

func styleToDoc(s model.TravelStyle) travelStyleDoc {
	interests := s.Interests
	if interests == nil {
		interests = []string{}
	}
	return travelStyleDoc{Interests: interests, Pace: s.Pace, Spend: s.Spend}
}

func userToDoc(u model.User) userDoc {
	d := userDoc{
		DisplayName:     u.DisplayName,
		PhotoURL:        u.PhotoURL,
		HomeRegion:      u.HomeRegion,
		HomeCurrency:    u.HomeCurrency,
		Units:           unitsToDoc(u.Units),
		Language:        u.Language,
		FormatLocale:    u.FormatLocale,
		RegionSource:    u.RegionSource,
		PassportCountry: u.PassportCountry,
		TravelStyle:     styleToDoc(u.TravelStyle),
		PaymentHandles:  handlesToDoc(u.PaymentHandles),
		Notifications:   u.Notifications,
		OnboardedAt:     u.OnboardedAt,
	}
	if u.HomeCity != nil {
		d.HomeCity = &homeCityDoc{Name: u.HomeCity.Name}
	}
	return d
}

func userFromDoc(id string, d userDoc) model.User {
	u := model.User{
		ID:              id,
		DisplayName:     d.DisplayName,
		PhotoURL:        d.PhotoURL,
		HomeRegion:      d.HomeRegion,
		HomeCurrency:    d.HomeCurrency,
		Units:           model.Units(d.Units),
		Language:        d.Language,
		FormatLocale:    d.FormatLocale,
		RegionSource:    d.RegionSource,
		PassportCountry: d.PassportCountry,
		TravelStyle:     model.TravelStyle{Interests: d.TravelStyle.Interests, Pace: d.TravelStyle.Pace, Spend: d.TravelStyle.Spend},
		PaymentHandles:  handlesFromDoc(d.PaymentHandles),
		Notifications:   d.Notifications,
		OnboardedAt:     d.OnboardedAt,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
	if len(u.TravelStyle.Interests) == 0 {
		u.TravelStyle.Interests = nil
	}
	if d.HomeCity != nil {
		u.HomeCity = &model.HomeCity{Name: d.HomeCity.Name}
	}
	return u
}

type destinationDoc struct {
	Name          string `firestore:"name"`
	CountryCode   string `firestore:"countryCode"`
	TimeZone      string `firestore:"timeZone"`
	LocalCurrency string `firestore:"localCurrency,omitempty"`
}

type flexibleDoc struct {
	Days  int    `firestore:"days"`
	Month string `firestore:"month,omitempty"`
}

type datesDoc struct {
	Start    string       `firestore:"start,omitempty"`
	End      string       `firestore:"end,omitempty"`
	Flexible *flexibleDoc `firestore:"flexible,omitempty"`
}

type moneyDoc struct {
	AmountMinor int64  `firestore:"amountMinor"`
	Currency    string `firestore:"currency"`
}

type budgetDoc struct {
	Scope  string   `firestore:"scope"`
	Amount moneyDoc `firestore:"amount"`
}

type planDoc struct {
	Unlocked bool   `firestore:"unlocked"`
	Source   string `firestore:"source,omitempty"`
}

type usageDoc struct {
	BestOrderRuns     int `firestore:"bestOrderRuns"`
	ForwardedBookings int `firestore:"forwardedBookings"`
	Files             int `firestore:"files"`
	Photos            int `firestore:"photos"`
}

type tripDoc struct {
	Title                string           `firestore:"title"`
	OwnerUID             string           `firestore:"ownerUid"`
	MemberUIDs           []string         `firestore:"memberUids"`
	Destinations         []destinationDoc `firestore:"destinations"`
	DestinationCountries []string         `firestore:"destinationCountries"`
	Dates                datesDoc         `firestore:"dates"`
	TimeZone             string           `firestore:"timeZone"`
	SettlementCurrency   string           `firestore:"settlementCurrency"`
	Budget               *budgetDoc       `firestore:"budget,omitempty"`
	Status               string           `firestore:"status"`
	Plan                 planDoc          `firestore:"plan"`
	Usage                usageDoc         `firestore:"usage"`
	EndedAt              *time.Time       `firestore:"endedAt,omitempty"`
	CreatedAt            time.Time        `firestore:"createdAt,serverTimestamp"`
	UpdatedAt            time.Time        `firestore:"updatedAt,serverTimestamp"`
}

func destinationsToDoc(in []model.Destination) []destinationDoc {
	out := make([]destinationDoc, len(in))
	for i, d := range in {
		out[i] = destinationDoc(d)
	}
	return out
}

func datesToDoc(d model.TripDates) datesDoc {
	out := datesDoc{Start: d.Start, End: d.End}
	if d.Flexible != nil {
		out.Flexible = &flexibleDoc{Days: d.Flexible.Days, Month: d.Flexible.Month}
	}
	return out
}

func budgetToDoc(b model.Budget) *budgetDoc {
	return &budgetDoc{Scope: b.Scope, Amount: moneyDoc(b.Amount)}
}

// countriesOf lists each destination country once, sorted.
func countriesOf(ds []model.Destination) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, d := range ds {
		if !seen[d.CountryCode] {
			seen[d.CountryCode] = true
			out = append(out, d.CountryCode)
		}
	}
	sort.Strings(out)
	return out
}

func tripToDoc(t model.Trip) tripDoc {
	d := tripDoc{
		Title:                t.Title,
		OwnerUID:             t.OwnerUID,
		MemberUIDs:           t.MemberUIDs,
		Destinations:         destinationsToDoc(t.Destinations),
		DestinationCountries: countriesOf(t.Destinations),
		Dates:                datesToDoc(t.Dates),
		TimeZone:             t.TimeZone,
		SettlementCurrency:   t.SettlementCurrency,
		Status:               string(t.Status),
		Plan:                 planDoc(t.Plan),
		Usage:                usageDoc(t.Usage),
		EndedAt:              t.EndedAt,
	}
	if t.Budget != nil {
		d.Budget = budgetToDoc(*t.Budget)
	}
	return d
}

func tripFromDoc(id string, d tripDoc) model.Trip {
	t := model.Trip{
		ID:                   id,
		Title:                d.Title,
		OwnerUID:             d.OwnerUID,
		MemberUIDs:           d.MemberUIDs,
		DestinationCountries: d.DestinationCountries,
		Dates:                model.TripDates{Start: d.Dates.Start, End: d.Dates.End},
		TimeZone:             d.TimeZone,
		SettlementCurrency:   d.SettlementCurrency,
		Status:               model.TripStatus(d.Status),
		Plan:                 model.TripPlan(d.Plan),
		Usage:                model.TripUsage(d.Usage),
		EndedAt:              d.EndedAt,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
	}
	for _, x := range d.Destinations {
		t.Destinations = append(t.Destinations, model.Destination(x))
	}
	if d.Dates.Flexible != nil {
		t.Dates.Flexible = &model.FlexibleDates{Days: d.Dates.Flexible.Days, Month: d.Dates.Flexible.Month}
	}
	if d.Budget != nil {
		t.Budget = &model.Budget{Scope: d.Budget.Scope, Amount: model.Money(d.Budget.Amount)}
	}
	return t
}

type memberDoc struct {
	UID            string             `firestore:"uid"`
	Role           string             `firestore:"role"`
	Status         string             `firestore:"status"`
	DisplayName    string             `firestore:"displayName"`
	PhotoURL       string             `firestore:"photoUrl,omitempty"`
	PaymentHandles []paymentHandleDoc `firestore:"paymentHandles"`
	InvitedVia     string             `firestore:"invitedVia,omitempty"`
	JoinedAt       time.Time          `firestore:"joinedAt,serverTimestamp"`
}

func memberToDoc(m model.Member) memberDoc {
	return memberDoc{
		UID:            m.UID,
		Role:           string(m.Role),
		Status:         string(m.Status),
		DisplayName:    m.DisplayName,
		PhotoURL:       m.PhotoURL,
		PaymentHandles: handlesToDoc(m.PaymentHandles),
		InvitedVia:     m.InvitedVia,
	}
}

func memberFromDoc(d memberDoc) model.Member {
	return model.Member{
		UID:            d.UID,
		Role:           model.Role(d.Role),
		Status:         model.MemberStatus(d.Status),
		DisplayName:    d.DisplayName,
		PhotoURL:       d.PhotoURL,
		PaymentHandles: handlesFromDoc(d.PaymentHandles),
		InvitedVia:     d.InvitedVia,
		JoinedAt:       d.JoinedAt,
	}
}
