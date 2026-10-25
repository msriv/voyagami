package server

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestRecap(t *testing.T) {
	trips := newFakeTrips()
	stops := &fakeStops{stops: map[string]model.Stop{}}
	journal := &fakeJournal{byID: map[string]model.JournalEntry{}}
	v := tokenVerifier{"owner": "u-owner", "other": "u-other", "editor": "u-editor"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithStops(stops), WithJournal(journal))
	var trip api.Trip
	if err := json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip); err != nil {
		t.Fatal(err)
	}
	trips.members[trip.Id]["u-editor"] = model.Member{UID: "u-editor", Role: model.RoleEditor, Status: model.MemberActive}
	path := "/v1/trips/" + trip.Id + "/recap"

	add := func(token, date, text, visibility string) {
		body := `{"text":"` + text + `","visibility":"` + visibility + `"}`
		if rec := call(h, "PUT", "/v1/trips/"+trip.Id+"/journal/"+date, token, body); rec.Code != 200 {
			t.Fatalf("journal: %d %s", rec.Code, rec.Body)
		}
	}
	add("owner", "2027-02-01", "Crew day one", "crew")
	add("editor", "2027-02-01", "Secret", "private")
	add("owner", "2027-02-02", "Day two", "crew")

	var got api.Recap
	rec := call(h, "GET", path, "owner", "")
	if rec.Code != 200 {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Crew < 1 || len(got.Highlights) != 2 || got.FavouriteDay == nil || got.FavouriteDay.Date != "2027-02-01" {
		t.Errorf("owner view: %+v", got)
	}
	if got.SettledUp != nil || got.TargetMinor != nil {
		t.Errorf("parts of off features should be absent: %+v", got)
	}

	// The editor's private entry shows for the editor, not for the owner.
	for token, want := range map[string]int{"owner": 2, "editor": 3} {
		var r api.Recap
		_ = json.Unmarshal(call(h, "GET", path, token, "").Body.Bytes(), &r)
		if len(r.Highlights) != want {
			t.Errorf("%s sees %d highlights, want %d", token, len(r.Highlights), want)
		}
	}

	if c := call(h, "GET", path, "other", "").Code; c != 404 {
		t.Errorf("non-member: %d", c)
	}
	if c := call(h, "GET", path, "", "").Code; c != 401 {
		t.Errorf("signed out: %d", c)
	}
}

func TestRecapOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	if c := call(h, "GET", "/v1/trips/t/recap", "owner", "").Code; c != 503 {
		t.Fatalf("got %d, want 503", c)
	}
}
