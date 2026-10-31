package model

import "time"

// Support message categories.
const (
	SupportBilling = "billing"
	SupportAccount = "account"
	SupportBug     = "bug"
	SupportOther   = "other"
)

// MaxSupportText is the longest support message, in bytes.
const MaxSupportText = 4000

// Diagnostics is what the phone adds to a support message. None of it names the
// person or a place.
type Diagnostics struct {
	AppVersion  string
	OS          string
	OSVersion   string
	DeviceModel string
	Locale      string
	RequestID   string
}

// SupportMessage is a message from the Help screen.
type SupportMessage struct {
	ID          string
	UserID      string
	Category    string
	Text        string
	Diagnostics Diagnostics
	CreatedAt   time.Time
}

// Validate checks the message before it is stored.
func (m SupportMessage) Validate() error {
	var p problems
	switch m.Category {
	case SupportBilling, SupportAccount, SupportBug, SupportOther:
	default:
		p.addf("category must be billing, account, bug or other")
	}
	if m.Text == "" || len(m.Text) > MaxSupportText {
		p.addf("text must be 1 to %d characters", MaxSupportText)
	}
	d := m.Diagnostics
	if len(d.AppVersion) > 32 || len(d.OSVersion) > 32 || len(d.DeviceModel) > 64 || len(d.Locale) > 35 || len(d.RequestID) > 64 {
		p.addf("diagnostics fields are too long")
	}
	if d.OS != "" && d.OS != "ios" && d.OS != "android" {
		p.addf("diagnostics os must be ios or android")
	}
	return p.err()
}
