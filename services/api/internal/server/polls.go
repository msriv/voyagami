package server

import (
	"context"
	"errors"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// WithPolls turns on the poll routes. They also need WithTrips; adding a
// winner to the itinerary also needs WithStops.
func WithPolls(s repo.Polls) Option { return func(h *handlers) { h.polls = s } }

const msgNoPolls = "Polls are not available right now."

func (h *handlers) pollsOff() bool { return h.trips == nil || h.polls == nil }

func unavailablePolls(ctx context.Context) api.UnavailableJSONResponse {
	return api.UnavailableJSONResponse(errBody(ctx, codeUnavailable, msgNoPolls))
}

func noSuchPoll(ctx context.Context) api.NotFoundJSONResponse {
	return api.NotFoundJSONResponse(errBody(ctx, codeNotFound, "No such poll."))
}

// pollView is the API shape of a poll for one caller. A poll past its closing
// time reads as closed, with the result worked out from the votes so far.
func pollView(p model.Poll, votes []model.Vote, uid string, now time.Time) api.Poll {
	status := p.Status
	result := p.Result
	if status == model.PollOpen && !p.Open(now) {
		status = model.PollClosed
		r, _ := model.Tally(p, votes, "")
		result = &r
	}
	counts, _ := model.Tally(p, votes, "") // live counts
	out := api.Poll{
		Id: p.ID, Question: p.Question, Multi: p.Multi, Status: api.PollStatus(status), CreatedBy: p.CreatedBy,
		MyVote: []string{}, Voters: len(votes), Options: make([]api.PollOption, len(p.Options)),
	}
	if status == model.PollClosed && p.Result != nil {
		counts = *p.Result // the frozen counts of a poll closed by hand
	}
	for i, o := range p.Options {
		out.Options[i] = api.PollOption{Id: o.ID, Label: o.Label, Votes: counts.Counts[o.ID]}
		if o.Place != nil {
			pl := placeTo(*o.Place)
			out.Options[i].Place = &pl
		}
	}
	for _, v := range votes {
		if v.UID == uid {
			out.MyVote = v.OptionIDs
		}
	}
	if !p.ClosesAt.IsZero() {
		t := p.ClosesAt
		out.ClosesAt = &t
	}
	if !p.ClosedAt.IsZero() {
		t := p.ClosedAt
		out.ClosedAt = &t
	}
	if result != nil {
		out.Result = &api.PollResult{DecidedBy: api.PollResultDecidedBy(result.DecidedBy), OptionId: strPtr(result.OptionID)}
	}
	return out
}

// ListPolls returns the trip's polls with live counts.
func (h *handlers) ListPolls(ctx context.Context, req api.ListPollsRequestObject) (api.ListPollsResponseObject, error) {
	if h.pollsOff() {
		return api.ListPolls503JSONResponse{UnavailableJSONResponse: unavailablePolls(ctx)}, nil
	}
	uid, _, deny, err := h.crewRole(ctx, "ListPolls", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ListPolls404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	all, err := h.polls.List(ctx, req.TripId)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := api.PollList{Polls: make([]api.Poll, len(all))}
	for i, p := range all {
		votes, err := h.polls.Votes(ctx, req.TripId, p.ID)
		if err != nil {
			return nil, err
		}
		out.Polls[i] = pollView(p, votes, uid, now)
	}
	return api.ListPolls200JSONResponse(out), nil
}

// CreatePoll starts a poll. Owners and editors only.
func (h *handlers) CreatePoll(ctx context.Context, req api.CreatePollRequestObject) (api.CreatePollResponseObject, error) {
	if h.pollsOff() {
		return api.CreatePoll503JSONResponse{UnavailableJSONResponse: unavailablePolls(ctx)}, nil
	}
	uid, role, deny, err := h.crewRole(ctx, "CreatePoll", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.CreatePoll404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if role == model.RoleViewer {
		return api.CreatePoll403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	b := req.Body
	p := model.Poll{ID: model.NewID(), Question: b.Question, CreatedBy: uid, Multi: b.Multi != nil && *b.Multi}
	if b.ClosesAt != nil {
		p.ClosesAt = *b.ClosesAt
		if !p.ClosesAt.After(time.Now()) {
			return api.CreatePoll400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, "closesAt must be in the future.")}, nil
		}
	}
	for _, o := range b.Options {
		opt := model.PollOption{ID: model.NewID(), Label: o.Label}
		if o.Place != nil {
			pl := placeFrom(*o.Place)
			opt.Place = &pl
		}
		p.Options = append(p.Options, opt)
	}
	created, err := h.polls.Create(ctx, req.TripId, p)
	if err != nil {
		if msg, ok := badRequest(err); ok {
			return api.CreatePoll400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	return api.CreatePoll201JSONResponse(pollView(created, nil, uid, time.Now())), nil
}

// VotePoll stores or changes the caller's vote while the poll is open.
func (h *handlers) VotePoll(ctx context.Context, req api.VotePollRequestObject) (api.VotePollResponseObject, error) {
	if h.pollsOff() {
		return api.VotePoll503JSONResponse{UnavailableJSONResponse: unavailablePolls(ctx)}, nil
	}
	uid, _, deny, err := h.crewRole(ctx, "VotePoll", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.VotePoll404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	err = h.polls.Vote(ctx, req.TripId, req.PollId, model.Vote{UID: uid, OptionIDs: req.Body.OptionIds}, time.Now())
	switch {
	case errors.Is(err, repo.ErrNotFound):
		return api.VotePoll404JSONResponse{NotFoundJSONResponse: noSuchPoll(ctx)}, nil
	case errors.Is(err, repo.ErrConflict):
		return api.VotePoll409JSONResponse{ConflictJSONResponse: api.ConflictJSONResponse(errBody(ctx, codeConflict, "This poll is closed."))}, nil
	case err != nil:
		if msg, ok := badRequest(err); ok {
			return api.VotePoll400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
		}
		return nil, err
	}
	p, votes, err := h.pollWithVotes(ctx, req.TripId, req.PollId)
	if err != nil {
		return nil, err
	}
	return api.VotePoll200JSONResponse(pollView(p, votes, uid, time.Now())), nil
}

func (h *handlers) pollWithVotes(ctx context.Context, tripID, pollID string) (model.Poll, []model.Vote, error) {
	p, err := h.polls.Get(ctx, tripID, pollID)
	if err != nil {
		return model.Poll{}, nil, err
	}
	votes, err := h.polls.Votes(ctx, tripID, pollID)
	return p, votes, err
}

// ClosePoll ends a poll and can put the winning place on the itinerary.
func (h *handlers) ClosePoll(ctx context.Context, req api.ClosePollRequestObject) (api.ClosePollResponseObject, error) {
	if h.pollsOff() {
		return api.ClosePoll503JSONResponse{UnavailableJSONResponse: unavailablePolls(ctx)}, nil
	}
	uid, role, deny, err := h.crewRole(ctx, "ClosePoll", req.TripId)
	if err != nil {
		return nil, err
	}
	if d, ok := deny.(api.NotFoundJSONResponse); ok {
		return api.ClosePoll404JSONResponse{NotFoundJSONResponse: d}, nil
	}
	if role == model.RoleViewer {
		return api.ClosePoll403JSONResponse{ForbiddenJSONResponse: errForbidden(ctx)}, nil
	}
	bad := func(msg string) (api.ClosePollResponseObject, error) {
		return api.ClosePoll400JSONResponse{BadRequestJSONResponse: errInvalid(ctx, msg)}, nil
	}
	chosen, date := strOr(req.Body.OptionId), strOr(req.Body.AddToDate)
	var trip model.Trip
	if date != "" {
		if h.stops == nil {
			return bad("The itinerary is not available right now.")
		}
		if !model.ValidDate(date) {
			return bad("addToDate must be YYYY-MM-DD.")
		}
		if trip, err = h.trips.Get(ctx, req.TripId); err != nil {
			return nil, err
		}
		if msg := dateOutsideTrip(trip, date); msg != "" {
			return bad(msg)
		}
	}
	before, err := h.polls.Get(ctx, req.TripId, req.PollId)
	if errors.Is(err, repo.ErrNotFound) {
		return api.ClosePoll404JSONResponse{NotFoundJSONResponse: noSuchPoll(ctx)}, nil
	}
	if err != nil {
		return nil, err
	}
	closed, err := h.polls.Close(ctx, req.TripId, req.PollId, chosen, time.Now())
	switch {
	case errors.Is(err, repo.ErrNotFound):
		return api.ClosePoll404JSONResponse{NotFoundJSONResponse: noSuchPoll(ctx)}, nil
	case errors.Is(err, model.ErrPollNoWinner):
		return api.ClosePoll409JSONResponse{ConflictJSONResponse: api.ConflictJSONResponse(errBody(ctx, codeConflict, "There is no clear winner. Pick one."))}, nil
	case err != nil:
		if msg, ok := badRequest(err); ok {
			return bad(msg)
		}
		return nil, err
	}
	votes, err := h.polls.Votes(ctx, req.TripId, req.PollId)
	if err != nil {
		return nil, err
	}
	out := api.PollClosed{Poll: pollView(closed, votes, uid, time.Now())}
	// Closing again returns the poll but never adds a second stop.
	if date == "" || closed.Result == nil || before.Status == model.PollClosed {
		return api.ClosePoll200JSONResponse(out), nil
	}
	for _, o := range closed.Options {
		if o.ID != closed.Result.OptionID || o.Place == nil {
			continue
		}
		stop := model.Stop{ID: model.NewID(), TripID: req.TripId, Date: date, TimeZone: trip.TimeZone, Place: *o.Place}
		if stop.Order, err = h.orderFor(ctx, req.TripId, date, "", ""); err != nil {
			return nil, err
		}
		created, err := h.stops.Create(ctx, stop)
		if err != nil {
			return nil, err
		}
		s := stopFrom(created)
		out.Stop = &s
	}
	return api.ClosePoll200JSONResponse(out), nil
}
