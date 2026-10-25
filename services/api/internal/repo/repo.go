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
