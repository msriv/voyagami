package model

// Action is something a crew member may try to do on a trip. Every handler
// asks Can instead of comparing roles, so the matrix in docs/security.md has
// one home in the code.
type Action int

const (
	// ActionEdit adds or changes shared trip content: stops, tasks, bookings,
	// expenses, polls, meet points, and the trip's name and dates.
	ActionEdit Action = iota + 1
	// ActionManageTrip changes the status (archive) and the budget.
	ActionManageTrip
	// ActionManageCrew changes roles, removes people and makes invites.
	ActionManageCrew
	// ActionModerate changes other people's work: any expense, any comment, and
	// settlements between other people.
	ActionModerate
	// ActionJournal writes the caller's own journal. Viewers read only.
	ActionJournal
)

// Can says whether a member with this role may do the action. Reading is not
// listed: every active member reads, and handlers check membership first.
func Can(role Role, a Action) bool {
	switch a {
	case ActionEdit, ActionJournal:
		return role == RoleOwner || role == RoleEditor
	case ActionManageTrip, ActionManageCrew, ActionModerate:
		return role == RoleOwner
	}
	return false
}
