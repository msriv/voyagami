package model

import "time"

// StampKind is one of the five starter stamps on Home.
type StampKind string

// The five starter stamps, in the order Home shows them.
const (
	StampCreateTrip   StampKind = "createTrip"
	StampInviteCrew   StampKind = "inviteCrew"
	StampAddPlaces    StampKind = "addPlaces"
	StampLogExpense   StampKind = "logExpense"
	StampWriteJournal StampKind = "writeJournal"
)

// StampKinds lists the starter stamps in display order.
var StampKinds = []StampKind{StampCreateTrip, StampInviteCrew, StampAddPlaces, StampLogExpense, StampWriteJournal}

// PlacesForStamp is how many places a person adds to earn the Add 3 places stamp.
const PlacesForStamp = 3

// EarnedStamp is a stamp a person has earned.
type EarnedStamp struct {
	Kind     StampKind
	EarnedAt time.Time
}
