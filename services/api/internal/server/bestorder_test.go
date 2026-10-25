package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
)

// addAt adds a pin at a latitude on Hanoi's longitude, so latitude is distance.
func (r *stopRig) addAt(date, name string, lat float64, extra string) api.Stop {
	body := `{"date":"` + date + `","place":{"name":"` + name + `","lat":` + ftoa(lat) + `,"lng":105.85}` + extra + `}`
	rec := call(r.h, "POST", r.path(""), "owner", body)
	var s api.Stop
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if rec.Code != 201 {
		r.t.Fatalf("add %s: %d %s", name, rec.Code, rec.Body)
	}
	return s
}

func ftoa(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func (r *stopRig) suggest(token, date string) (int, api.BestOrder) {
	rec := call(r.h, "POST", "/v1/trips/"+r.tripID+"/days/"+date+"/best-order", token, "")
	var b api.BestOrder
	_ = json.Unmarshal(rec.Body.Bytes(), &b)
	return rec.Code, b
}

func (r *stopRig) setOrder(token, date string, ids []string) int {
	body, _ := json.Marshal(api.DayOrder{StopIds: ids})
	return call(r.h, "PUT", "/v1/trips/"+r.tripID+"/days/"+date+"/order", token, string(body)).Code
}

func TestBestOrderSuggestAcceptAndUndo(t *testing.T) {
	r := newStopRig(t)
	const day = "2027-02-02"
	// Latitudes zig-zag: A and C are near each other, B and D near each other.
	a := r.addAt(day, "A", 21.00, "")
	b := r.addAt(day, "B", 21.10, "")
	c := r.addAt(day, "C", 21.01, "")
	d := r.addAt(day, "D", 21.09, "")
	original := []string{a.Id, b.Id, c.Id, d.Id}

	code, sug := r.suggest("viewer", day) // any member may ask
	if code != 200 || !sug.Changed || !sug.Approx || sug.AfterMin >= sug.BeforeMin {
		t.Fatalf("suggest: %d %+v", code, sug)
	}
	if strings.Join(sug.StopIds, ",") != strings.Join([]string{a.Id, c.Id, d.Id, b.Id}, ",") {
		t.Fatalf("order %v", sug.StopIds)
	}
	r.expect(day, "A", "B", "C", "D") // asking changed nothing

	// Accept, then undo by sending the earlier order back.
	if code := r.setOrder("editor", day, sug.StopIds); code != 200 {
		t.Fatalf("accept: %d", code)
	}
	r.expect(day, "A", "C", "D", "B")
	if code := r.setOrder("editor", day, original); code != 200 {
		t.Fatalf("undo: %d", code)
	}
	r.expect(day, "A", "B", "C", "D")

	// A second look at an already-short day finds nothing to change.
	r.setOrder("owner", day, sug.StopIds)
	if _, again := r.suggest("owner", day); again.Changed || again.BeforeMin != again.AfterMin {
		t.Fatalf("already best: %+v", again)
	}
}

func TestBestOrderKeepsTimedAndCatalogueStops(t *testing.T) {
	r := newStopRig(t)
	const day = "2027-02-03"
	r.addAt(day, "A", 21.00, "")
	r.addAt(day, "Lunch", 21.10, `,"startLocal":"12:30"`)
	r.addAt(day, "C", 21.01, "")
	cat := call(r.h, "POST", r.path(""), "owner", `{"date":"`+day+`","place":{"placeId":"hanoi-1"}}`)
	if cat.Code != 201 {
		t.Fatalf("catalogue stop: %d", cat.Code)
	}
	_, sug := r.suggest("owner", day)
	// Slot 1 is the timed lunch and slot 3 the catalogue stop; both stay put.
	var all api.StopList
	_ = json.Unmarshal(call(r.h, "GET", r.path(""), "owner", "").Body.Bytes(), &all)
	if sug.StopIds[1] != all.Stops[1].Id || sug.StopIds[3] != all.Stops[3].Id {
		t.Fatalf("pinned stops moved: %v", sug.StopIds)
	}
}

func TestDayOrderRulesAndErrors(t *testing.T) {
	r := newStopRig(t)
	const day = "2027-02-04"
	a := r.addAt(day, "A", 21.00, "")
	b := r.addAt(day, "B", 21.05, "")
	other := r.addAt("2027-02-05", "Elsewhere", 21.2, "")
	tests := []struct {
		name  string
		token string
		ids   []string
		want  int
	}{
		{"viewer cannot set the order", "viewer", []string{b.Id, a.Id}, 403},
		{"non-member", "other", []string{b.Id, a.Id}, 404},
		{"a stop is missing", "owner", []string{a.Id}, 400},
		{"a stop is repeated", "owner", []string{a.Id, a.Id}, 400},
		{"a stop from another day", "owner", []string{a.Id, other.Id}, 400},
		{"unknown stop", "owner", []string{a.Id, "nope"}, 400},
		{"editor reorders", "editor", []string{b.Id, a.Id}, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.setOrder(tt.token, day, tt.ids); got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
	r.expect(day, "B", "A")
	if code, _ := r.suggest("other", day); code != 404 {
		t.Errorf("non-member suggest: %d", code)
	}
	if code, _ := r.suggest("owner", "feb-3"); code != 400 {
		t.Errorf("bad date: %d", code)
	}
	if code, _ := r.suggest("", day); code != 401 {
		t.Errorf("signed out: %d", code)
	}
	if code, sug := r.suggest("owner", "2027-02-01"); code != 200 || sug.Changed || len(sug.StopIds) != 0 {
		t.Errorf("empty day: %d %+v", code, sug)
	}
}
