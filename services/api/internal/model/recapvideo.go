package model

import "time"

// RecapVideoState is where a recap video render is.
type RecapVideoState string

// The states of a render job.
const (
	RecapQueued    RecapVideoState = "queued"
	RecapRendering RecapVideoState = "rendering"
	RecapReady     RecapVideoState = "ready"
	RecapFailed    RecapVideoState = "failed"
)

// RecapVideo is one render job for a trip's recap video.
type RecapVideo struct {
	ID          string
	TripID      string
	RequestedBy string
	State       RecapVideoState
	Progress    int // 0 to 100
	// MaxSeconds and Watermark are fixed when the job is made, from the trip's
	// plan at that time. MaxSeconds is -1 for no cap.
	MaxSeconds int
	Watermark  bool
	Object     string // storage object name once ready
	Error      string // plain English, once failed
	Attempts   int
	CreatedAt  time.Time
	StartedAt  time.Time
}

// Active reports whether the job is still waiting or running.
func (v RecapVideo) Active() bool { return v.State == RecapQueued || v.State == RecapRendering }
