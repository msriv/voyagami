package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

// fakePolls is an in-memory repo.Polls that follows the same rules as the real one.
type fakePolls struct {
	polls map[string]model.Poll
	votes map[string]map[string]model.Vote
}

func newFakePolls() *fakePolls {
	return &fakePolls{polls: map[string]model.Poll{}, votes: map[string]map[string]model.Vote{}}
}

func (f *fakePolls) Create(_ context.Context, _ string, p model.Poll) (model.Poll, error) {
	p.Status, p.CreatedAt = model.PollOpen, time.Now()
	if err := p.Validate(); err != nil {
		return model.Poll{}, err
	}
	f.polls[p.ID], f.votes[p.ID] = p, map[string]model.Vote{}
	return p, nil
}

func (f *fakePolls) Get(_ context.Context, _, id string) (model.Poll, error) {
	p, ok := f.polls[id]
	if !ok {
		return model.Poll{}, repo.ErrNotFound
	}
	return p, nil
}

func (f *fakePolls) List(context.Context, string) ([]model.Poll, error) {
	var out []model.Poll
	for _, p := range f.polls {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (f *fakePolls) Vote(_ context.Context, _, id string, v model.Vote, now time.Time) error {
	p, ok := f.polls[id]
	if !ok {
		return repo.ErrNotFound
	}
	if !p.Open(now) {
		return repo.ErrConflict
	}
	if err := v.Validate(p); err != nil {
		return err
	}
	f.votes[id][v.UID] = v
	return nil
}

func (f *fakePolls) Votes(_ context.Context, _, id string) ([]model.Vote, error) {
	var out []model.Vote
	for _, v := range f.votes[id] {
		out = append(out, v)
	}
	return out, nil
}

func (f *fakePolls) Close(ctx context.Context, tripID, id, chosen string, now time.Time) (model.Poll, error) {
	p, ok := f.polls[id]
	if !ok {
		return model.Poll{}, repo.ErrNotFound
	}
	if p.Status == model.PollClosed {
		return p, nil
	}
	votes, _ := f.Votes(ctx, tripID, id)
	res, err := model.Tally(p, votes, chosen)
	if err != nil {
		return model.Poll{}, err
	}
	if res.OptionID == "" {
		return model.Poll{}, model.ErrPollNoWinner
	}
	p.Status, p.ClosedAt, p.Result = model.PollClosed, now, &res
	f.polls[id] = p
	return p, nil
}

func TestPolls(t *testing.T) {
	trips, stops := newFakeTrips(), &fakeStops{stops: map[string]model.Stop{}}
	v := tokenVerifier{"owner": "u-owner", "editor": "u-editor", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithStops(stops), WithPolls(newFakePolls()))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	base := "/v1/trips/" + trip.Id + "/polls"

	create := `{"question":"Dinner?","options":[{"label":"Pho","place":{"name":"Pho 10","lat":21.03,"lng":105.85}},{"label":"Bun cha"}]}`
	for tok, want := range map[string]int{"viewer": 403, "other": 404} {
		if rec := call(h, "POST", base, tok, create); rec.Code != want {
			t.Errorf("%s creates: %d", tok, rec.Code)
		}
	}
	for _, bad := range []string{
		`{"question":"","options":[{"label":"a"},{"label":"b"}]}`,
		`{"question":"One option","options":[{"label":"a"}]}`,
		`{"question":"Past","closesAt":"2020-01-01T00:00:00Z","options":[{"label":"a"},{"label":"b"}]}`,
	} {
		if rec := call(h, "POST", base, "editor", bad); rec.Code != 400 {
			t.Errorf("bad create %s: %d", bad, rec.Code)
		}
	}
	rec := call(h, "POST", base, "editor", create)
	var poll api.Poll
	if err := json.Unmarshal(rec.Body.Bytes(), &poll); err != nil || rec.Code != 201 || poll.Status != "open" || len(poll.Options) != 2 || poll.Options[0].Place == nil || len(poll.MyVote) != 0 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	pho, bun := poll.Options[0].Id, poll.Options[1].Id
	one := base + "/" + poll.Id

	// Everyone on the trip can vote, viewers too; a stranger cannot.
	if rec := call(h, "POST", one+"/vote", "other", `{"optionIds":["`+pho+`"]}`); rec.Code != 404 {
		t.Errorf("stranger votes: %d", rec.Code)
	}
	for _, bad := range []string{`{"optionIds":[]}`, `{"optionIds":["nope"]}`, `{"optionIds":["` + pho + `","` + bun + `"]}`} {
		if rec := call(h, "POST", one+"/vote", "viewer", bad); rec.Code != 400 {
			t.Errorf("bad vote %s: %d", bad, rec.Code)
		}
	}
	if rec := call(h, "POST", base+"/nope/vote", "viewer", `{"optionIds":["`+pho+`"]}`); rec.Code != 404 {
		t.Errorf("unknown poll: %d", rec.Code)
	}
	vote := func(tok, opt string) api.Poll {
		t.Helper()
		rec := call(h, "POST", one+"/vote", tok, `{"optionIds":["`+opt+`"]}`)
		var p api.Poll
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || rec.Code != 200 {
			t.Fatalf("vote: %d %s", rec.Code, rec.Body)
		}
		return p
	}
	vote("viewer", pho)
	p := vote("editor", bun)
	if p.Voters != 2 || p.Options[0].Votes != 1 || p.Options[1].Votes != 1 || len(p.MyVote) != 1 || p.MyVote[0] != bun {
		t.Fatalf("after two votes: %+v", p)
	}
	// Changing a vote moves it; it does not add one.
	if p = vote("editor", pho); p.Voters != 2 || p.Options[0].Votes != 2 || p.Options[1].Votes != 0 {
		t.Fatalf("after change: %+v", p)
	}

	if rec := call(h, "POST", one+"/close", "viewer", `{}`); rec.Code != 403 {
		t.Errorf("viewer closes: %d", rec.Code)
	}
	if rec := call(h, "POST", one+"/close", "owner", `{"optionId":"nope"}`); rec.Code != 400 {
		t.Errorf("close with foreign option: %d", rec.Code)
	}
	if rec := call(h, "POST", one+"/close", "owner", `{"addToDate":"2030-01-01"}`); rec.Code != 400 {
		t.Errorf("date outside the trip: %d", rec.Code)
	}
	rec = call(h, "POST", one+"/close", "owner", `{"addToDate":"2027-02-02"}`)
	var closed api.PollClosed
	if err := json.Unmarshal(rec.Body.Bytes(), &closed); err != nil || rec.Code != 200 || closed.Poll.Status != "closed" ||
		closed.Poll.Result == nil || *closed.Poll.Result.OptionId != pho || closed.Poll.Result.DecidedBy != "vote" || closed.Stop == nil || closed.Stop.Date != "2027-02-02" {
		t.Fatalf("close: %d %s", rec.Code, rec.Body)
	}
	if len(stops.stops) != 1 {
		t.Errorf("stops: %d", len(stops.stops))
	}
	// Closing again is harmless and adds no second stop; voting is over.
	if rec := call(h, "POST", one+"/close", "owner", `{"addToDate":"2027-02-02"}`); rec.Code != 200 || len(stops.stops) != 1 {
		t.Errorf("close twice: %d, stops %d", rec.Code, len(stops.stops))
	}
	if rec := call(h, "POST", one+"/vote", "viewer", `{"optionIds":["`+bun+`"]}`); rec.Code != 409 {
		t.Errorf("vote after close: %d", rec.Code)
	}
	rec = call(h, "GET", base, "viewer", "")
	var list api.PollList
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != 200 || len(list.Polls) != 1 || list.Polls[0].Status != "closed" || list.Polls[0].MyVote[0] != pho {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
}

func TestPollTieNeedsAPick(t *testing.T) {
	trips := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner", "editor": "u-editor"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithPolls(newFakePolls()))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	base := "/v1/trips/" + trip.Id + "/polls"

	var poll api.Poll
	_ = json.Unmarshal(call(h, "POST", base, "owner", `{"question":"Where?","multi":true,"options":[{"label":"A"},{"label":"B"}]}`).Body.Bytes(), &poll)
	a, b := poll.Options[0].Id, poll.Options[1].Id
	one := base + "/" + poll.Id
	if rec := call(h, "POST", one+"/close", "owner", `{}`); rec.Code != 409 {
		t.Errorf("no votes: %d", rec.Code)
	}
	call(h, "POST", one+"/vote", "owner", `{"optionIds":["`+a+`"]}`)
	call(h, "POST", one+"/vote", "editor", `{"optionIds":["`+b+`"]}`)
	if rec := call(h, "POST", one+"/close", "owner", `{}`); rec.Code != 409 {
		t.Errorf("tie: %d", rec.Code)
	}
	rec := call(h, "POST", one+"/close", "owner", `{"optionId":"`+b+`"}`)
	var closed api.PollClosed
	if err := json.Unmarshal(rec.Body.Bytes(), &closed); err != nil || rec.Code != 200 || closed.Poll.Result.DecidedBy != "owner" || *closed.Poll.Result.OptionId != b {
		t.Fatalf("owner pick: %d %s", rec.Code, rec.Body)
	}
}

func TestPollEndsOnItsOwnTime(t *testing.T) {
	p := model.Poll{ID: "p", Question: "q", Status: model.PollOpen, ClosesAt: time.Now().Add(-time.Minute),
		Options: []model.PollOption{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}}
	votes := []model.Vote{{UID: "u1", OptionIDs: []string{"a"}}}
	view := pollView(p, votes, "u1", time.Now())
	if view.Status != "closed" || view.Result == nil || *view.Result.OptionId != "a" || view.Options[0].Votes != 1 || view.MyVote[0] != "a" {
		t.Fatalf("expired poll: %+v", view)
	}
	if err := (model.Vote{UID: "u1", OptionIDs: []string{"a"}}).Validate(p); err != nil {
		t.Fatal(err)
	}
}
