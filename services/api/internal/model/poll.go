package model

import (
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Limits on polls.
const (
	MaxPollQuestion = 200
	MaxPollLabel    = 100
	MinPollOptions  = 2
	MaxPollOptions  = 10
)

// Poll statuses and how a result was decided.
const (
	PollOpen     = "open"
	PollClosed   = "closed"
	DecidedVote  = "vote"
	DecidedOwner = "owner"
)

// ErrPollNoWinner means a poll cannot close by vote because nobody led: no
// votes, or a tie. The closer must pick an option.
var ErrPollNoWinner = errors.New("poll has no clear winner")

// PollOption is one choice. Place is set when the option is somewhere to go.
type PollOption struct {
	ID    string
	Label string
	Place *PlaceRef
}

// PollResult is what a closed poll decided. OptionID is empty when a poll
// ended on its own time with no clear winner.
type PollResult struct {
	OptionID  string
	Counts    map[string]int
	DecidedBy string // "vote" or "owner"
}

// Poll is a question the crew votes on.
type Poll struct {
	ID        string
	Question  string
	Options   []PollOption
	Multi     bool // several options per vote; otherwise exactly one
	Status    string
	ClosesAt  time.Time // zero means it stays open until someone closes it
	ClosedAt  time.Time
	Result    *PollResult
	CreatedBy string
	CreatedAt time.Time
}

// Vote is one member's choice. There is one per member, so voting again replaces it.
type Vote struct {
	UID       string
	OptionIDs []string
	VotedAt   time.Time
}

// Validate checks a poll before it is stored.
func (p Poll) Validate() error {
	var pr problems
	if p.ID == "" || p.CreatedBy == "" {
		pr.addf("id and createdBy are required")
	}
	if q := strings.TrimSpace(p.Question); q == "" || utf8.RuneCountInString(q) > MaxPollQuestion {
		pr.addf("question is required and at most %d characters", MaxPollQuestion)
	}
	if n := len(p.Options); n < MinPollOptions || n > MaxPollOptions {
		pr.addf("a poll needs %d to %d options", MinPollOptions, MaxPollOptions)
	}
	seen := map[string]bool{}
	for _, o := range p.Options {
		if o.ID == "" || seen[o.ID] {
			pr.addf("option ids must be set and different")
		}
		seen[o.ID] = true
		if l := strings.TrimSpace(o.Label); l == "" || utf8.RuneCountInString(l) > MaxPollLabel {
			pr.addf("each option needs a label of at most %d characters", MaxPollLabel)
		}
		if o.Place != nil {
			o.Place.validate(&pr)
		}
	}
	return pr.err()
}

// Open reports whether votes are still taken at the given time.
func (p Poll) Open(now time.Time) bool {
	return p.Status == PollOpen && (p.ClosesAt.IsZero() || now.Before(p.ClosesAt))
}

// Validate checks a vote against the poll it is for.
func (v Vote) Validate(p Poll) error {
	var pr problems
	n := len(v.OptionIDs)
	switch {
	case n == 0:
		pr.addf("pick at least one option")
	case !p.Multi && n != 1:
		pr.addf("pick exactly one option")
	}
	valid := map[string]bool{}
	for _, o := range p.Options {
		valid[o.ID] = true
	}
	seen := map[string]bool{}
	for _, id := range v.OptionIDs {
		if !valid[id] || seen[id] {
			pr.addf("options must belong to the poll and be picked once")
			break
		}
		seen[id] = true
	}
	return pr.err()
}

// Tally counts the votes. A person can only have voted for the poll's own
// options, so anything else is ignored. If chosen is set the result goes to
// that option, decided by the owner; otherwise the option with most votes
// wins, and a tie or no votes leaves OptionID empty.
func Tally(p Poll, votes []Vote, chosen string) (PollResult, error) {
	counts := make(map[string]int, len(p.Options))
	for _, o := range p.Options {
		counts[o.ID] = 0
	}
	for _, v := range votes {
		for _, id := range v.OptionIDs {
			if _, ok := counts[id]; ok {
				counts[id]++
			}
		}
	}
	res := PollResult{Counts: counts, DecidedBy: DecidedVote}
	if chosen != "" {
		if _, ok := counts[chosen]; !ok {
			return PollResult{}, &ValidationError{Problems: []string{"optionId must belong to the poll"}}
		}
		res.OptionID, res.DecidedBy = chosen, DecidedOwner
		return res, nil
	}
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return counts[ids[i]] > counts[ids[j]] })
	if top := counts[ids[0]]; top > 0 && (len(ids) == 1 || counts[ids[1]] < top) {
		res.OptionID = ids[0]
	}
	return res, nil
}
