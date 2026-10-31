package model

import "time"

// ConsentState is one choice about one purpose. A person who was never asked
// is unset; granting and denying can both be changed later.
type ConsentState string

// The three states.
const (
	ConsentUnset   ConsentState = "unset"
	ConsentGranted ConsentState = "granted"
	ConsentDenied  ConsentState = "denied"
)

// Consents holds the choices a person made, one per purpose.
type Consents struct {
	Location      ConsentState
	Notifications ConsentState
	Photos        ConsentState
	Analytics     ConsentState
	UpdatedAt     time.Time
}

// ConsentsPatch changes only the purposes that are set.
type ConsentsPatch struct {
	Location, Notifications, Photos, Analytics *ConsentState
}

// Validate checks that every set purpose is granted or denied.
func (p ConsentsPatch) Validate() error {
	var pr problems
	for name, s := range map[string]*ConsentState{"location": p.Location, "notifications": p.Notifications, "photos": p.Photos, "analytics": p.Analytics} {
		if s != nil && *s != ConsentGranted && *s != ConsentDenied {
			pr.addf("%s must be granted or denied", name)
		}
	}
	return pr.err()
}

// Empty reports whether the patch changes nothing.
func (p ConsentsPatch) Empty() bool {
	return p.Location == nil && p.Notifications == nil && p.Photos == nil && p.Analytics == nil
}

// Apply returns c with the patch applied.
func (c Consents) Apply(p ConsentsPatch, now time.Time) Consents {
	for dst, src := range map[*ConsentState]*ConsentState{&c.Location: p.Location, &c.Notifications: p.Notifications, &c.Photos: p.Photos, &c.Analytics: p.Analytics} {
		if src != nil {
			*dst = *src
		}
	}
	c.UpdatedAt = now
	return c
}

// UnsetConsents is what a person who was never asked has.
func UnsetConsents() Consents {
	return Consents{Location: ConsentUnset, Notifications: ConsentUnset, Photos: ConsentUnset, Analytics: ConsentUnset}
}
