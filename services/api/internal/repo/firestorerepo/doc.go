// Package firestorerepo implements internal/repo on Firestore, using the
// collections and fields described in docs/data-model.md.
package firestorerepo

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/msriv/voyagami/services/api/internal/repo"
)

const (
	usersCollection   = "users"
	tripsCollection   = "trips"
	membersCollection = "members"
	// phoneIndexCollection maps a keyed phone hash to a user ID. It is server-only.
	phoneIndexCollection = "phoneIndex"
	sessionsCollection   = "sessions"
)

// mapErr turns Firestore errors into the repo's sentinel errors. Errors that
// are already repo or validation errors pass through.
func mapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repo.ErrNotFound) || errors.Is(err, repo.ErrAlreadyExists) || errors.Is(err, repo.ErrForbidden) {
		return err
	}
	switch status.Code(err) {
	case codes.NotFound:
		return repo.ErrNotFound
	case codes.AlreadyExists:
		return repo.ErrAlreadyExists
	}
	return fmt.Errorf("%s: %w", op, err)
}

func isNotFound(err error) bool { return status.Code(err) == codes.NotFound }
