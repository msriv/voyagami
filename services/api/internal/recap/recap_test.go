package recap

import (
	"strings"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/model"
)

func pin(id, date, order string, lat, lng float64) model.Stop {
	return model.Stop{ID: id, Date: date, Order: order, Place: model.PlaceRef{Name: id, Lat: lat, Lng: lng}}
}

func TestBuild(t *testing.T) {
	trip := model.Trip{
		MemberUIDs: []string{"a", "b", "c", "d"}, SettlementCurrency: "VND",
		Dates:  model.TripDates{Start: "2026-11-19", End: "2026-11-27"},
		Budget: &model.Budget{Scope: "perPerson", Amount: model.Money{AmountMinor: 1000, Currency: "VND"}},
	}
	stops := []model.Stop{
		pin("a", "2026-11-19", "a0", 0, 0),
		pin("b", "2026-11-19", "a1", 0, 1), // about 111.2 km east of a
		{ID: "cat", Date: "2026-11-20", Order: "a0", Place: model.PlaceRef{PlaceID: "p1"}},
	}
	entries := []model.JournalEntry{
		{Date: "2026-11-21", AuthorUID: "a", Text: "Fog on the bay."},
		{Date: "2026-11-21", AuthorUID: "b", Text: strings.Repeat("x", 200)},
		{Date: "2026-11-20", AuthorUID: "a", Text: ""},
	}
	got := Build(Input{Trip: trip, Stops: stops, Entries: entries, SpentMinor: 3500, Balances: map[string]int64{"a": 0, "b": 0}})

	if got.Days != 9 || got.Stops != 3 || got.Crew != 4 || got.Currency != "VND" {
		t.Errorf("basics: %+v", got)
	}
	if got.DistanceMeters < 111000 || got.DistanceMeters > 111400 {
		t.Errorf("distance = %d", got.DistanceMeters)
	}
	if got.TargetMinor == nil || *got.TargetMinor != 4000 || *got.RemainingMinor != 500 {
		t.Errorf("budget: %v %v", got.TargetMinor, got.RemainingMinor)
	}
	if got.SettledUp == nil || !*got.SettledUp {
		t.Errorf("settled: %v", got.SettledUp)
	}
	if got.FavouriteDay == nil || got.FavouriteDay.Date != "2026-11-21" || got.FavouriteDay.Count != 2 {
		t.Errorf("favourite: %+v", got.FavouriteDay)
	}
	if len(got.Highlights) != 2 || got.Highlights[0].Text != "Fog on the bay." || !strings.HasSuffix(got.Highlights[1].Text, "…") {
		t.Errorf("highlights: %+v", got.Highlights)
	}
}

func TestBuildEmptyAndFlexible(t *testing.T) {
	trip := model.Trip{SettlementCurrency: "INR", Dates: model.TripDates{Flexible: &model.FlexibleDates{Days: 3}}}
	got := Build(Input{Trip: trip, Stops: []model.Stop{pin("a", "2026-01-01", "a0", 1, 1), pin("b", "2026-01-03", "a0", 1, 1)}})
	if got.Days != 2 || got.DistanceMeters != 0 || got.FavouriteDay != nil || got.SettledUp != nil || got.TargetMinor != nil || len(got.Highlights) != 0 {
		t.Errorf("%+v", got)
	}
}

func TestBuildCapsHighlightsAndUnsettled(t *testing.T) {
	var es []model.JournalEntry
	for i := 0; i < 8; i++ {
		es = append(es, model.JournalEntry{Date: "2026-01-0" + string(rune('1'+i)), AuthorUID: "a", Text: "t"})
	}
	got := Build(Input{Trip: model.Trip{}, Entries: es, Balances: map[string]int64{"a": 5, "b": -5}})
	if len(got.Highlights) != MaxHighlights || *got.SettledUp {
		t.Errorf("%+v", got)
	}
}
