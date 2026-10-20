// Package repo defines how the API reads and writes stored data. Handlers and
// services depend on these interfaces; internal/repo/firestorerepo implements
// them, and a later Cloud SQL move (P3) replaces only that package.
package repo

import (
	"context"
	"errors"

	"github.com/msriv/voyagami/services/api/internal/model"
)

var (
	// ErrNotFound means the document does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists means a create hit an existing document.
	ErrAlreadyExists = errors.New("already exists")
	// ErrForbidden means the change breaks a trip rule, such as removing the owner.
	ErrForbidden = errors.New("not allowed")
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
