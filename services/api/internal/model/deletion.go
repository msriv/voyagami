package model

import "time"

// DeletionGrace is how long a person has to change their mind after asking to
// delete their account.
const DeletionGrace = 30 * 24 * time.Hour

// Deletion is a request to delete an account. Personal data is removed at DeleteOn.
type Deletion struct {
	UserID      string
	RequestedAt time.Time
	DeleteOn    time.Time
}
