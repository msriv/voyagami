package ranking

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden file")

type golden struct {
	Places  []Place `json:"places"`
	Style   Style   `json:"style"`
	Anchor  [2]float64
	Minute  int
	Ranking []struct {
		ID      string   `json:"id"`
		Score   float64  `json:"score"`
		Reasons []Signal `json:"reasons"`
	} `json:"ranking"`
}

// TestGolden pins the order and scores for a small Hanoi list. If a weight
// change moves them, run `go test ./internal/ranking -update`, read the diff,
// and keep it only if the new order is the one you want.
func TestGolden(t *testing.T) {
	const file = "testdata/hanoi.golden.json"
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var g golden
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	got := Rank(g.Places, Context{Style: g.Style, AnchorLat: g.Anchor[0], AnchorLng: g.Anchor[1], HasAnchor: true, MinuteOfDay: g.Minute})
	g.Ranking = g.Ranking[:0]
	for _, s := range got {
		g.Ranking = append(g.Ranking, struct {
			ID      string   `json:"id"`
			Score   float64  `json:"score"`
			Reasons []Signal `json:"reasons"`
		}{s.Place.ID, float64(int(s.Score*10000+0.5)) / 10000, s.Reasons})
	}
	out, _ := json.MarshalIndent(g, "", "  ")
	out = append(out, '\n')
	if *update {
		if err := os.WriteFile(file, out, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	if string(out) != string(raw) {
		t.Fatalf("ranking changed; rerun with -update and read the diff.\n got: %s", out)
	}
}

func ctx() Context { return Context{MinuteOfDay: -1} }

func TestWeightsAddUp(t *testing.T) {
	sum := WeightStyle + WeightQuality + WeightDistance + WeightPopularity + WeightTime + WeightData
	if sum < 0.999999 || sum > 1.000001 {
		t.Fatalf("weights add up to %v, want 1", sum)
	}
}

func TestFewReviewsDoNotBeatManyGoodOnes(t *testing.T) {
	few := Place{ID: "few", Rating: 5, Reviews: 1}
	many := Place{ID: "many", Rating: 4.6, Reviews: 800}
	if quality(few) >= quality(many) {
		t.Fatalf("1 review at 5.0 (%v) beat 800 at 4.6 (%v)", quality(few), quality(many))
	}
}

func TestClosedPlacesAreDropped(t *testing.T) {
	c := Context{MinuteOfDay: 23 * 60}
	got := Rank([]Place{
		{ID: "closed", OpensMin: 8 * 60, ClosesMin: 17 * 60},
		{ID: "late", OpensMin: 18 * 60, ClosesMin: 2 * 60}, // past midnight
		{ID: "unknown"},
	}, c)
	if len(got) != 2 || got[0].Place.ID == "closed" || got[1].Place.ID == "closed" {
		t.Fatalf("got %v", got)
	}
}

func TestCloserWinsAllElseEqual(t *testing.T) {
	c := Context{AnchorLat: 21.03, AnchorLng: 105.85, HasAnchor: true, MinuteOfDay: -1}
	got := Rank([]Place{
		{ID: "far", Lat: 21.10, Lng: 105.85, HasCoord: true},
		{ID: "near", Lat: 21.031, Lng: 105.85, HasCoord: true},
	}, c)
	if got[0].Place.ID != "near" {
		t.Fatalf("got %s first", got[0].Place.ID)
	}
}

func TestVarietySpreadsKinds(t *testing.T) {
	got := Rank([]Place{
		{ID: "t1", Kind: "temple", Rating: 4.9, Reviews: 900},
		{ID: "t2", Kind: "temple", Rating: 4.8, Reviews: 900},
		{ID: "t3", Kind: "temple", Rating: 4.7, Reviews: 900},
		{ID: "m1", Kind: "market", Rating: 4.5, Reviews: 900},
	}, ctx())
	if got[1].Place.ID != "m1" && got[2].Place.ID != "m1" {
		t.Fatalf("market should interleave, got %s %s %s %s", got[0].Place.ID, got[1].Place.ID, got[2].Place.ID, got[3].Place.ID)
	}
}

func TestTiesBreakByID(t *testing.T) {
	got := Rank([]Place{{ID: "b"}, {ID: "a"}, {ID: "c"}}, ctx())
	if got[0].Place.ID != "a" || got[1].Place.ID != "b" || got[2].Place.ID != "c" {
		t.Fatalf("got %v", got)
	}
}

func TestCrewSavesNeedEnoughCrewsToBeAReason(t *testing.T) {
	few := Rank([]Place{{ID: "p", CrewSaves: CrewMinimum - 1}}, ctx())[0]
	many := Rank([]Place{{ID: "p", CrewSaves: 90}}, ctx())[0]
	for _, r := range few.Reasons {
		if r == SignalPopularity {
			t.Fatal("saves below the minimum must not be a reason")
		}
	}
	found := false
	for _, r := range many.Reasons {
		found = found || r == SignalPopularity
	}
	if !found {
		t.Fatalf("90 saves should be a reason, got %v (parts %v)", many.Reasons, many.Parts)
	}
}
