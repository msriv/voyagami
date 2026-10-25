package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Limits on comments.
const (
	MaxCommentBody     = 2000
	MaxCommentMentions = 10
)

// What a comment can be about.
const (
	TargetStop = "stop"
	TargetTrip = "trip"
)

// Comment is one note in a thread on a stop or on the trip.
type Comment struct {
	ID         string
	TargetType string // TargetStop or TargetTrip
	TargetID   string // the stop's ID, or the trip's ID
	AuthorUID  string
	Body       string
	Mentions   []string // members the author tagged
	CreatedAt  time.Time
	EditedAt   time.Time // zero if never edited
}

// Validate checks a comment before it is stored.
func (c Comment) Validate() error {
	var p problems
	if c.ID == "" || c.AuthorUID == "" || c.TargetID == "" {
		p.addf("id, authorUid and targetId are required")
	}
	if c.TargetType != TargetStop && c.TargetType != TargetTrip {
		p.addf("targetType must be stop or trip")
	}
	commentTextProblems(&p, c.Body, c.Mentions)
	return p.err()
}

// ValidateCommentText checks the parts of a comment a person can change.
func ValidateCommentText(body string, mentions []string) error {
	var p problems
	commentTextProblems(&p, body, mentions)
	return p.err()
}

func commentTextProblems(p *problems, body string, mentions []string) {
	if b := strings.TrimSpace(body); b == "" || utf8.RuneCountInString(b) > MaxCommentBody {
		p.addf("body is required and at most %d characters", MaxCommentBody)
	}
	if len(mentions) > MaxCommentMentions {
		p.addf("at most %d mentions", MaxCommentMentions)
	}
}
