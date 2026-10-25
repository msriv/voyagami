package model

// Change names one item that changed on a trip. It holds no content, so the
// feed leaks nothing a member could not fetch.
type Change struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Op   string `json:"op"` // "upsert" or "delete"
}
