// Package entitlements decides what a person may do on their plan.
//
// Every limit is checked here, on the server. The app only reflects the answer.
// The numbers come from the Plans Spec board. Dev runs with every limit off
// (see ForEnv); Free holds the real values.
package entitlements

import "sort"

// Feature is something a plan can limit.
type Feature string

// Counted features have a number limit. Flag features are on or off.
const (
	// UpcomingTrips counts a person's upcoming trips. Past trips are not counted.
	UpcomingTrips Feature = "upcomingTrips"
	// BestOrder counts route-optimisation runs on a trip.
	BestOrder Feature = "bestOrder"
	// ForwardedBookings counts bookings imported from forwarded email on a trip.
	ForwardedBookings Feature = "forwardedBookings"
	// Files counts files attached to a trip.
	Files Feature = "files"
	// JournalPhotos counts photos in a trip's journal.
	JournalPhotos Feature = "journalPhotos"
	// AskDrafts counts a person's Ask drafts in the current month.
	AskDrafts Feature = "askDrafts"
	// RecapSeconds is the longest recap video a trip may make. Free is capped and
	// watermarked; Plus, a Trip Pass or a Plus crew member lifts both. It is a
	// length, not a counter, so Used is always 0. Use Recap for the full answer.
	RecapSeconds Feature = "recapSeconds"

	// TopPicks is how many recommendations a person sees per category. It is a
	// cap on a list, not a counter, so Used is always 0. Use PickLimit.
	TopPicks Feature = "topPicks"

	// MultiCurrency lets a trip hold expenses in more than one currency.
	MultiCurrency Feature = "multiCurrency"
	// LiveCrewMap shows the crew's live locations all day.
	LiveCrewMap Feature = "liveCrewMap"
	// ReceiptScans reads receipts and splits them by item.
	ReceiptScans Feature = "receiptScans"
	// OfflineMaps downloads maps for use without a signal.
	OfflineMaps Feature = "offlineMaps"
	// GmailImport finds bookings in the person's inbox.
	GmailImport Feature = "gmailImport"
	// DarkMode switches the app to its dark theme.
	DarkMode Feature = "darkMode"

	// The always-free features. No plan, limit table or override can close them.
	JoinTrip    Feature = "joinTrip"
	SettleUp    Feature = "settleUp"
	MeetPoints  Feature = "meetPoints"
	DataExport  Feature = "dataExport"
	OfflinePlan Feature = "offlinePlan"
)

// Unlimited as a limit means no cap. For a flag feature, 0 means Plus only and
// anything else means open to everyone.
const Unlimited = -1

// scope says who a Plus or Trip Pass unlocks a feature for.
type scope int

const (
	// scopeFree is never limited.
	scopeFree scope = iota
	// scopeUser unlocks for the person who has Plus, or anyone on a trip with a pass.
	scopeUser
	// scopeTrip unlocks for everyone on the trip when any member has Plus or the trip has a pass.
	scopeTrip
)

var scopes = map[Feature]scope{
	UpcomingTrips:     scopeUser,
	AskDrafts:         scopeUser,
	ReceiptScans:      scopeUser,
	OfflineMaps:       scopeUser,
	GmailImport:       scopeUser,
	DarkMode:          scopeUser,
	BestOrder:         scopeTrip,
	RecapSeconds:      scopeTrip,
	ForwardedBookings: scopeTrip,
	Files:             scopeTrip,
	JournalPhotos:     scopeUser,
	TopPicks:          scopeUser,
	MultiCurrency:     scopeTrip,
	LiveCrewMap:       scopeTrip,
	JoinTrip:          scopeFree,
	SettleUp:          scopeFree,
	MeetPoints:        scopeFree,
	DataExport:        scopeFree,
	OfflinePlan:       scopeFree,
}

// Limits holds what a person without Plus gets, per feature.
type Limits map[Feature]int

// Free returns the Free plan values from the Plans Spec board.
func Free() Limits {
	return Limits{
		UpcomingTrips:     2,
		BestOrder:         1,
		ForwardedBookings: 5,
		Files:             3,
		JournalPhotos:     30,
		TopPicks:          5,
		AskDrafts:         3,
		RecapSeconds:      30,
		MultiCurrency:     0,
		LiveCrewMap:       0,
		ReceiptScans:      0,
		OfflineMaps:       0,
		GmailImport:       0,
		DarkMode:          0,
	}
}

// NoLimits returns a table with every limit off, for dev.
func NoLimits() Limits {
	l := Free()
	for f := range l {
		l[f] = Unlimited
	}
	return l
}

// ForEnv returns the limits for an APP_ENV value: none in dev, Free elsewhere.
func ForEnv(env string) Limits {
	if env == "dev" {
		return NoLimits()
	}
	return Free()
}

// User is what the service needs to know about the person asking.
type User struct {
	// Plus is true while the person has an active Plus subscription or trial.
	Plus bool
	// Used holds the person's own counters: UpcomingTrips and AskDrafts.
	Used map[Feature]int
}

// Trip is what the service needs to know about the trip. A nil Trip means the
// check is not about one trip.
type Trip struct {
	// Pass is true while the trip has an active Trip Pass.
	Pass bool
	// MemberPlus is true when any member of the trip has Plus.
	MemberPlus bool
	// Used holds the trip's counters: BestOrder, ForwardedBookings, Files and JournalPhotos.
	Used map[Feature]int
}

// Decision is the answer to a check.
type Decision struct {
	Allowed bool
	// Limit is the cap, or Unlimited. For a flag feature it is 0 when closed.
	Limit int
	// Used is how many the person or trip has already used.
	Used int
	// NeedsPlus is true when the answer is no and Plus (or a Trip Pass) would change it.
	NeedsPlus bool
	// ReadOnly is true when the person or trip is already over the cap, which
	// happens after a downgrade. The existing content stays and can be read, but
	// not changed, until the count is back under the cap or Plus returns.
	ReadOnly bool
}

// Service answers entitlement checks.
type Service struct {
	free Limits
}

// New returns a service that gives Free people the given limits.
func New(free Limits) *Service { return &Service{free: free} }

// Check says whether the user may add one more of a counted feature, or use a
// flag feature. Unknown features are refused.
func (s *Service) Check(user User, trip *Trip, f Feature) Decision {
	sc, known := scopes[f]
	if !known {
		return Decision{}
	}
	if sc == scopeFree {
		return Decision{Allowed: true, Limit: Unlimited}
	}
	used := user.Used[f]
	if trip != nil && countedPerTrip(f) {
		used = trip.Used[f]
	}
	if unlocked(f, sc, user, trip) {
		return Decision{Allowed: true, Limit: Unlimited, Used: used}
	}
	limit, ok := s.free[f]
	if !ok {
		return Decision{Used: used, NeedsPlus: true}
	}
	switch {
	case limit == Unlimited:
		return Decision{Allowed: true, Limit: Unlimited, Used: used}
	case isFlag(f):
		return Decision{Allowed: limit != 0, Limit: limit, Used: used, NeedsPlus: limit == 0}
	default:
		ok := used < limit
		return Decision{Allowed: ok, Limit: limit, Used: used, NeedsPlus: !ok, ReadOnly: used > limit}
	}
}

func unlocked(f Feature, sc scope, user User, trip *Trip) bool {
	if user.Plus {
		return true
	}
	if trip == nil {
		return false
	}
	if trip.Pass && !accountOnly(f) {
		return true
	}
	return sc == scopeTrip && trip.MemberPlus
}

// accountOnly is true for features that belong to the person, not to a trip, so a Trip Pass never opens them.
func accountOnly(f Feature) bool {
	switch f {
	case UpcomingTrips, AskDrafts, DarkMode:
		return true
	}
	return false
}

// countedPerTrip is true for counters kept on the trip rather than the person.
func countedPerTrip(f Feature) bool {
	switch f {
	case BestOrder, ForwardedBookings, Files, JournalPhotos:
		return true
	}
	return false
}

func isFlag(f Feature) bool {
	switch f {
	case MultiCurrency, LiveCrewMap, ReceiptScans, OfflineMaps, GmailImport, DarkMode:
		return true
	}
	return false
}

// All lists every feature the service knows, in a stable order.
func All() []Feature {
	all := make([]Feature, 0, len(scopes))
	for f := range scopes {
		all = append(all, f)
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	return all
}

// RecapPolicy is what a trip's recap video may be.
type RecapPolicy struct {
	// MaxSeconds is the longest recap, or Unlimited.
	MaxSeconds int
	// Watermark is true when the video carries the Voyagami watermark.
	Watermark bool
}

// Recap says how long a recap video may be and whether it is watermarked. The
// whole crew gets the Plus answer when any member has Plus or the trip has a pass.
func (s *Service) Recap(user User, trip *Trip) RecapPolicy {
	d := s.Check(user, trip, RecapSeconds)
	return RecapPolicy{MaxSeconds: d.Limit, Watermark: d.Limit != Unlimited}
}

// PickLimit is how many recommendations the person may see per category, or
// Unlimited. Free sees the top 5; Plus, or a Trip Pass on the trip, sees all.
func (s *Service) PickLimit(user User, trip *Trip) int {
	return s.Check(user, trip, TopPicks).Limit
}
