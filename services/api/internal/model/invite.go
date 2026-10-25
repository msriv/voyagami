package model

import "time"

// Invite is a link that lets someone join a trip. Only the SHA-256 hash of the
// secret token is stored; the ID is that hash.
type Invite struct {
	ID        string
	TripID    string
	Role      Role // editor or viewer, never owner
	CreatedBy string
	CreatedAt time.Time
	ExpiresAt time.Time // zero means it never expires
	MaxUses   int       // zero means unlimited
	UseCount  int
	RevokedAt time.Time // zero means not revoked
}

// Usable reports whether someone can still join with the invite at the given time.
func (i Invite) Usable(now time.Time) bool {
	switch {
	case !i.RevokedAt.IsZero():
		return false
	case !i.ExpiresAt.IsZero() && !now.Before(i.ExpiresAt):
		return false
	case i.MaxUses > 0 && i.UseCount >= i.MaxUses:
		return false
	}
	return true
}

// Validate checks an invite before it is stored.
func (i Invite) Validate() error {
	var p problems
	if i.ID == "" || i.TripID == "" || i.CreatedBy == "" {
		p.addf("id, tripId and createdBy are required")
	}
	if i.Role != RoleEditor && i.Role != RoleViewer {
		p.addf("role must be editor or viewer")
	}
	if i.MaxUses < 0 {
		p.addf("maxUses cannot be negative")
	}
	return p.err()
}
