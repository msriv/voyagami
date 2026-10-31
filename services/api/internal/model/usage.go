package model

// UsageCounter names a counter in TripUsage.
type UsageCounter string

// The counters the free-plan limits read.
const (
	UsageBestOrderRuns     UsageCounter = "bestOrderRuns"
	UsageForwardedBookings UsageCounter = "forwardedBookings"
	UsageFiles             UsageCounter = "files"
	UsagePhotos            UsageCounter = "photos"
)
