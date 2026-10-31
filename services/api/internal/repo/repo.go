// Package repo defines how the API reads and writes stored data. Handlers and
// services depend on these interfaces; internal/repo/firestorerepo implements
// them, and a later Cloud SQL move (P3) replaces only that package.
package repo

import (
	"context"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/model"
)

var (
	// ErrNotFound means the document does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists means a create hit an existing document.
	ErrAlreadyExists = errors.New("already exists")
	// ErrForbidden means the change breaks a trip rule, such as removing the owner.
	ErrForbidden = errors.New("not allowed")
	// ErrConflict means the change is not possible in the current state, such
	// as voting in a closed poll.
	ErrConflict = errors.New("conflict")
	// ErrTokenReuse means a refresh token that was already replaced was
	// presented again. The session is revoked.
	ErrTokenReuse = errors.New("refresh token reused")
)

// Invalid input is reported as a *model.ValidationError.

// Users stores people's own profiles.
type Users interface {
	// Create stores a new user. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, u model.User) (model.User, error)
	// Get returns the user, or ErrNotFound.
	Get(ctx context.Context, id string) (model.User, error)
	// Update changes only the fields set in the patch.
	Update(ctx context.Context, id string, patch model.UserPatch) (model.User, error)
}

// TripUsage counts what a trip has used of the plan limits.
type TripUsage interface {
	// Add raises one counter on the trip by n, atomically.
	Add(ctx context.Context, tripID string, c model.UsageCounter, n int) error
}

// Stamps stores the starter stamps a person has earned (VOY-202).
type Stamps interface {
	// Award records a stamp the first time. It reports whether this call earned it.
	Award(ctx context.Context, uid string, kind model.StampKind, at time.Time) (bool, error)
	// AddPlace counts one place the person added and awards the Add 3 places
	// stamp when the count reaches model.PlacesForStamp. It reports whether this
	// call earned the stamp.
	AddPlace(ctx context.Context, uid string, at time.Time) (bool, error)
	// List returns the stamps the person has earned, in no set order.
	List(ctx context.Context, uid string) ([]model.EarnedStamp, error)
}

// Devices stores the phones a person is signed in on and their push tokens.
type Devices interface {
	// Put stores or refreshes a device, keeping one document per device ID.
	Put(ctx context.Context, uid string, d model.Device) error
	// Delete removes a device. Removing one that is not there is not an error.
	Delete(ctx context.Context, uid, deviceID string) error
	// List returns the person's devices.
	List(ctx context.Context, uid string) ([]model.Device, error)
}

// Trips stores trips and who is on them. Membership changes keep the trip's
// memberUids mirror in step inside one transaction.
type Trips interface {
	// Create stores a trip and its owner in one step. The owner's role is
	// forced to owner and their status to active.
	Create(ctx context.Context, t model.Trip, owner model.Member) (model.Trip, error)
	// Get returns the trip, or ErrNotFound.
	Get(ctx context.Context, id string) (model.Trip, error)
	// ListForMember returns the trips an active member is on, soonest first.
	// Trips without fixed dates come after the dated ones.
	ListForMember(ctx context.Context, uid string) ([]model.Trip, error)
	// Update changes only the fields set in the patch.
	Update(ctx context.Context, id string, patch model.TripPatch) (model.Trip, error)

	// Member returns one member of the trip, including people who left.
	Member(ctx context.Context, tripID, uid string) (model.Member, error)
	// Members returns everyone with a member document on the trip.
	Members(ctx context.Context, tripID string) ([]model.Member, error)
	// AddMember adds an editor or viewer. It returns ErrAlreadyExists if that
	// person is already active, and reactivates someone who left or was removed.
	AddMember(ctx context.Context, tripID string, m model.Member) error
	// SetMemberRole changes an active editor or viewer to editor or viewer.
	// The owner's role cannot be changed (ErrForbidden).
	SetMemberRole(ctx context.Context, tripID, uid string, role model.Role) error
	// EndMembership marks a member left or removed. The owner cannot be
	// removed (ErrForbidden).
	EndMembership(ctx context.Context, tripID, uid string, status model.MemberStatus) error
}

// Polls stores a trip's polls and the votes on them. The caller has already
// checked that the person may use the trip.
type Polls interface {
	// Create stores a poll. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, tripID string, p model.Poll) (model.Poll, error)
	// Get returns the poll, or ErrNotFound.
	Get(ctx context.Context, tripID, id string) (model.Poll, error)
	// List returns the trip's polls, newest first.
	List(ctx context.Context, tripID string) ([]model.Poll, error)
	// Vote stores or replaces the member's vote. It returns ErrNotFound for a
	// missing poll and ErrConflict if the poll is closed at now.
	Vote(ctx context.Context, tripID, pollID string, v model.Vote, now time.Time) error
	// Votes returns every vote on the poll, by voter.
	Votes(ctx context.Context, tripID, pollID string) ([]model.Vote, error)
	// Close counts the votes and ends the poll in one step. A non-empty chosen
	// decides the winner. Closing a closed poll returns it unchanged. With no
	// chosen and no clear winner it returns model.ErrPollNoWinner.
	Close(ctx context.Context, tripID, pollID, chosen string, now time.Time) (model.Poll, error)
}

// Comments stores the notes on a trip's stops. The caller has already checked
// that the person may use the trip and may change the comment.
type Comments interface {
	// Create stores a comment. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, tripID string, c model.Comment) (model.Comment, error)
	// Get returns the comment, or ErrNotFound.
	Get(ctx context.Context, tripID, id string) (model.Comment, error)
	// List returns the trip's comments, oldest first.
	List(ctx context.Context, tripID string) ([]model.Comment, error)
	// Edit replaces the body and mentions and stamps the edit time. It returns
	// ErrNotFound if the comment does not exist.
	Edit(ctx context.Context, tripID, id, body string, mentions []string, now time.Time) (model.Comment, error)
	// Delete removes a comment. Deleting a missing one is not an error.
	Delete(ctx context.Context, tripID, id string) error
}

// MeetPoints stores a trip's meet points. The caller has already checked that
// the person may use the trip.
type MeetPoints interface {
	// Create stores a meet point. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, tripID string, m model.MeetPoint) (model.MeetPoint, error)
	// Get returns the meet point, or ErrNotFound.
	Get(ctx context.Context, tripID, id string) (model.MeetPoint, error)
	// List returns the trip's meet points, soonest first, ones with no time last.
	List(ctx context.Context, tripID string) ([]model.MeetPoint, error)
	// Update changes only the fields set in the patch. It returns ErrNotFound
	// if the meet point does not exist.
	Update(ctx context.Context, tripID, id string, p model.MeetPointPatch) (model.MeetPoint, error)
	// SetConfirmed adds or removes the person from those coming, and returns
	// the meet point. Doing it twice is harmless.
	SetConfirmed(ctx context.Context, tripID, id, uid string, coming bool) (model.MeetPoint, error)
	// Delete removes a meet point. Deleting a missing one is not an error.
	Delete(ctx context.Context, tripID, id string) error
}

// Presence stores the locations people are sharing on a trip. It is throwaway:
// entries end by themselves and are removed once over.
type Presence interface {
	// Put stores or replaces the person's shared location.
	Put(ctx context.Context, tripID string, s model.Presence) error
	// Get returns the person's entry, or ErrNotFound. It may already be over.
	Get(ctx context.Context, tripID, uid string) (model.Presence, error)
	// Delete stops the person's sharing. Stopping when not sharing is not an error.
	Delete(ctx context.Context, tripID, uid string) error
	// List returns locations still being shared at now, and removes the ones that ended.
	List(ctx context.Context, tripID string, now time.Time) ([]model.Presence, error)
}

// Invites stores trip invite links. Invites are looked up by the hash of
// their secret token.
type Invites interface {
	// Create stores an invite. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, i model.Invite) error
	// Get returns the invite, or ErrNotFound.
	Get(ctx context.Context, id string) (model.Invite, error)
	// List returns the trip's invites, newest first.
	List(ctx context.Context, tripID string) ([]model.Invite, error)
	// Revoke ends an invite that belongs to the trip. Revoking one that is
	// already revoked is not an error. It returns ErrNotFound if the invite
	// does not exist or belongs to another trip.
	Revoke(ctx context.Context, tripID, id string, now time.Time) error
	// Use counts one join and returns the invite. In one step it checks the
	// invite is usable at now, so two people cannot take the last use. It
	// returns ErrNotFound if the invite is missing, revoked, expired or used up.
	Use(ctx context.Context, id string, now time.Time) (model.Invite, error)
}

// Stops stores the places on a trip's days.
type Stops interface {
	// Create stores a new stop. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, s model.Stop) (model.Stop, error)
	// Get returns the stop, or ErrNotFound. Deleted stops are returned too.
	Get(ctx context.Context, tripID, stopID string) (model.Stop, error)
	// List returns the trip's stops that are not deleted, sorted by (date, order).
	List(ctx context.Context, tripID string) ([]model.Stop, error)
	// Update changes only the fields set in the patch.
	Update(ctx context.Context, tripID, stopID string, patch model.StopPatch) (model.Stop, error)
	// SetOrders rewrites the order key of several stops in one atomic step.
	// It returns ErrNotFound if any of the stops does not exist.
	SetOrders(ctx context.Context, tripID string, orders map[string]string) error
}

// Expenses stores a trip's expenses. The caller has already checked that the
// person may use the trip and may change the expense.
type Expenses interface {
	// Create stores an expense. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, tripID string, e model.Expense) (model.Expense, error)
	// Get returns the expense, or ErrNotFound if it is missing or deleted.
	Get(ctx context.Context, tripID, expenseID string) (model.Expense, error)
	// List returns expenses that are not deleted, newest date first.
	List(ctx context.Context, tripID string) ([]model.Expense, error)
	// Replace overwrites the expense's content and keeps who made it and when.
	// It returns ErrNotFound if it is missing or deleted.
	Replace(ctx context.Context, tripID string, e model.Expense) (model.Expense, error)
	// Delete marks the expense deleted. Deleting a missing one is not an error.
	Delete(ctx context.Context, tripID, expenseID string) error
}

// Settlements stores payments members have marked as paid.
type Settlements interface {
	// Create stores a payment. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, tripID string, s model.Settlement) (model.Settlement, error)
	// Get returns the payment, or ErrNotFound.
	Get(ctx context.Context, tripID, id string) (model.Settlement, error)
	// List returns the trip's payments, oldest first.
	List(ctx context.Context, tripID string) ([]model.Settlement, error)
}

// Bookings stores a trip's bookings. The caller has already checked that the
// person may use the trip.
type Bookings interface {
	// Create stores a booking. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, tripID string, b model.Booking) (model.Booking, error)
	// Get returns the booking, or ErrNotFound.
	Get(ctx context.Context, tripID, id string) (model.Booking, error)
	// List returns the trip's bookings, soonest start first, undated last.
	List(ctx context.Context, tripID string) ([]model.Booking, error)
	// Replace overwrites the booking's content and keeps who made it and when.
	// It returns ErrNotFound if it does not exist.
	Replace(ctx context.Context, tripID string, b model.Booking) (model.Booking, error)
	// Delete removes a booking. Deleting a missing one is not an error.
	Delete(ctx context.Context, tripID, id string) error
}

// Journal stores journal entries. It does not decide who may read a private
// entry; the caller filters.
type Journal interface {
	// Put creates the entry, or replaces its content if that person already
	// has one for the day. It keeps the original creation time.
	Put(ctx context.Context, tripID string, e model.JournalEntry) (model.JournalEntry, error)
	// List returns every entry on the trip, newest day first, then by author.
	List(ctx context.Context, tripID string) ([]model.JournalEntry, error)
	// Delete removes one person's entry for a day. Deleting a missing one is not an error.
	Delete(ctx context.Context, tripID, date, authorUID string) error
}

// Tasks stores a trip's to-dos. The caller has already checked that the
// person may use the trip.
type Tasks interface {
	// Create stores a task. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, tripID string, t model.Task) (model.Task, error)
	// List returns open tasks first, then done ones, oldest first within each.
	List(ctx context.Context, tripID string) ([]model.Task, error)
	// Update changes only the fields set in the patch. It returns ErrNotFound
	// if the task does not exist.
	Update(ctx context.Context, tripID, taskID string, p model.TaskPatch) (model.Task, error)
	// Delete removes a task. Deleting a missing task is not an error.
	Delete(ctx context.Context, tripID, taskID string) error
}

// Accounts links a phone number to a user without storing the number. The
// caller passes a keyed hash of the number, never the number itself.
type Accounts interface {
	// FindOrCreate returns the user linked to phoneHash. If there is none it
	// stores newUser and the link in one step and reports created.
	FindOrCreate(ctx context.Context, phoneHash string, newUser model.User) (u model.User, created bool, err error)
}

// Sessions stores sign-in sessions, one per device.
type Sessions interface {
	// Create stores a new session. It returns ErrAlreadyExists if the ID is taken.
	Create(ctx context.Context, s model.Session) error
	// Get returns the session, or ErrNotFound.
	Get(ctx context.Context, id string) (model.Session, error)
	// Rotate swaps the refresh hash and expiry if oldHash is the current hash.
	// It returns ErrNotFound for a missing, revoked or expired session. If
	// oldHash is not the current hash it revokes the session and returns
	// ErrTokenReuse.
	Rotate(ctx context.Context, id, oldHash, newHash string, expiresAt time.Time) error
	// Revoke ends the session. Revoking a missing or ended session is not an error.
	Revoke(ctx context.Context, id string) error
}

// Feed reports changes to a trip as they happen.
type Feed interface {
	// Watch calls emit for every change to the trip made after Watch starts,
	// until ctx ends. emit may be called from several goroutines. An error
	// means the watch broke and the caller should reconnect.
	Watch(ctx context.Context, tripID string, emit func(model.Change)) error
}
