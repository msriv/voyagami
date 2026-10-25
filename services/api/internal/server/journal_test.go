package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeJournal struct{ byID map[string]model.JournalEntry }

func (f *fakeJournal) Put(_ context.Context, _ string, e model.JournalEntry) (model.JournalEntry, error) {
	if err := e.Validate(); err != nil {
		return model.JournalEntry{}, err
	}
	f.byID[e.ID()] = e
	return e, nil
}

func (f *fakeJournal) List(context.Context, string) ([]model.JournalEntry, error) {
	var out []model.JournalEntry
	for _, e := range f.byID {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date > out[j].Date
		}
		return out[i].AuthorUID < out[j].AuthorUID
	})
	return out, nil
}

func (f *fakeJournal) Delete(_ context.Context, _, date, uid string) error {
	delete(f.byID, date+"_"+uid)
	return nil
}

func TestJournal(t *testing.T) {
	trips := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner", "editor": "u-editor", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithJournal(&fakeJournal{byID: map[string]model.JournalEntry{}}))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	base := "/v1/trips/" + trip.Id + "/journal"
	day := base + "/2027-02-02"

	rec := call(h, "PUT", day, "editor", `{"text":"Pho for breakfast.","mood":"happy","stopIds":["s1"]}`)
	var e api.JournalEntry
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if rec.Code != 200 || e.AuthorUid != "u-editor" || e.Visibility != "crew" || e.Mood == nil || *e.Mood != "happy" || len(e.StopIds) != 1 {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}
	// Saving again replaces it: still one entry for that person and day.
	if c := call(h, "PUT", day, "editor", `{"text":"Pho for breakfast, banh mi for lunch."}`).Code; c != 200 {
		t.Fatalf("second put: %d", c)
	}
	call(h, "PUT", day, "owner", `{"text":"Secret plans.","visibility":"private"}`)
	call(h, "PUT", base+"/2027-02-03", "owner", `{"text":"Day two."}`)

	list := func(token string) []api.JournalEntry {
		rec := call(h, "GET", base, token, "")
		var l api.JournalList
		if rec.Code != 200 {
			t.Fatalf("list %s: %d %s", token, rec.Code, rec.Body)
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &l)
		return l.Entries
	}
	got := list("viewer")
	if len(got) != 2 || got[0].Date != "2027-02-03" || got[1].AuthorUid != "u-editor" || !strings.HasPrefix(got[1].Text, "Pho for breakfast, banh mi") {
		t.Fatalf("viewer sees: %+v", got)
	}
	if got := list("owner"); len(got) != 3 {
		t.Fatalf("owner should see own private entry too: %+v", got)
	}
	// A private entry is hidden from everyone else, the owner included.
	call(h, "PUT", day, "editor", `{"text":"Just me.","visibility":"private"}`)
	for _, token := range []string{"viewer", "owner"} {
		for _, x := range list(token) {
			if x.Text == "Just me." && token != "editor" {
				t.Errorf("%s can read someone else's private entry", token)
			}
		}
	}

	for name, c := range map[string]struct {
		token, path, body string
		want              int
	}{
		"viewer writes":   {"viewer", day, `{"text":"hi"}`, 403},
		"outsider writes": {"other", day, `{"text":"hi"}`, 404},
		"bad date":        {"owner", base + "/3-Feb", `{"text":"hi"}`, 400},
		"bad mood":        {"owner", day, `{"text":"hi","mood":"furious"}`, 400},
		"bad visibility":  {"owner", day, `{"text":"hi","visibility":"public"}`, 400},
		"too long":        {"owner", day, `{"text":"` + strings.Repeat("a", model.MaxJournalText+1) + `"}`, 400},
		"too many stops":  {"owner", day, `{"text":"hi","stopIds":["1","2","3","4","5","6","7","8","9","10","11","12","13","14","15","16","17","18","19","20","21"]}`, 400},
	} {
		if got := call(h, "PUT", c.path, c.token, c.body).Code; got != c.want {
			t.Errorf("%s: got %d, want %d", name, got, c.want)
		}
	}
	if c := call(h, "GET", base, "other", "").Code; c != 404 {
		t.Errorf("outsider list: %d", c)
	}

	// Delete removes only your own entry.
	if c := call(h, "DELETE", day, "editor", "").Code; c != 204 {
		t.Errorf("delete: %d", c)
	}
	if c := call(h, "DELETE", day, "editor", "").Code; c != 204 {
		t.Errorf("delete again: %d", c)
	}
	if c := call(h, "DELETE", day, "viewer", "").Code; c != 403 {
		t.Errorf("viewer delete: %d", c)
	}
	for _, x := range list("owner") {
		if x.AuthorUid == "u-editor" {
			t.Errorf("editor's entry should be gone: %+v", x)
		}
	}
}

func TestJournalOff(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	if c := call(h, "GET", "/v1/trips/t/journal", "owner", "").Code; c != 503 {
		t.Errorf("got %d", c)
	}
}
