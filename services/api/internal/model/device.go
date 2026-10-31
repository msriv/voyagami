package model

import "time"

// Platform names the push service a device token belongs to.
type Platform string

// Supported platforms. iOS tokens go through FCM too (APNs via FCM).
const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

// Device is one phone a person is signed in on, with the token pushes go to.
type Device struct {
	ID         string
	Platform   Platform
	PushToken  string
	LastSeenAt time.Time
}

const maxTokenLen = 4096

// Validate checks the device before it is stored.
func (d Device) Validate() error {
	var p problems
	if d.ID == "" || len(d.ID) > 128 {
		p.addf("deviceId must be 1 to 128 characters")
	}
	if d.Platform != PlatformIOS && d.Platform != PlatformAndroid {
		p.addf("platform must be ios or android")
	}
	if d.PushToken == "" || len(d.PushToken) > maxTokenLen {
		p.addf("pushToken must be 1 to 4096 characters")
	}
	return p.err()
}
