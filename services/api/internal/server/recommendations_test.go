package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/entitlements"
	"github.com/msriv/voyagami/services/api/internal/model"
)

func TestRecommendations(t *testing.T) {
	// Two places in the same spot: one suits food lovers, one does not.
	noodles := model.CatalogPlace{ID: "noodles", Name: map[string]string{"en": "Bun cha"}, Lat: 21.03, Lng: 105.85, Group: model.GroupEat,
		Categories: []string{"noodles"}, Tags: []string{"foodTrails"}, Rating: 4.6, Reviews: 800}
	chain := model.CatalogPlace{ID: "chain", Name: map[string]string{"en": "Chain cafe"}, Lat: 21.03, Lng: 105.85, Group: model.GroupEat,
		Categories: []string{"cafe"}, Rating: 3.9, Reviews: 30}
	cat := &fakeCatalog{places: []model.NearbyPlace{{CatalogPlace: chain, Distance: 10}, {CatalogPlace: noodles, Distance: 10}}}
	profile := &fakeProfile{user: model.User{TravelStyle: model.TravelStyle{Interests: []string{"foodTrails"}, Spend: "comfort"}}}
	trips := newFakeTrips()
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner", "other": "u-other"},
		WithTrips(trips), WithStops(&fakeStops{stops: map[string]model.Stop{}}), WithCatalog(cat), WithProfile(profile))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	base := "/v1/trips/" + trip.Id + "/recommendations?category=eat"

	t.Run("no stops and no position is a 400", func(t *testing.T) {
		if got := call(h, "GET", base, "owner", "").Code; got != 400 {
			t.Fatalf("got %d", got)
		}
	})
	t.Run("a stop with a position gives the centre and the crew's style puts food first", func(t *testing.T) {
		if rec := call(h, "POST", "/v1/trips/"+trip.Id+"/stops", "owner", `{"date":"2027-02-02","place":{"name":"Hotel","lat":21.03,"lng":105.85}}`); rec.Code != 201 {
			t.Fatalf("stop: %d %s", rec.Code, rec.Body)
		}
		rec := call(h, "GET", base, "owner", "")
		if rec.Code != 200 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		var list api.RecommendationList
		_ = json.Unmarshal(rec.Body.Bytes(), &list)
		if len(list.Items) != 2 || list.Items[0].Place.Id != "noodles" || list.HasMore {
			t.Fatalf("items %+v", list.Items)
		}
		reasons := list.Items[0].Reasons
		if len(reasons) == 0 || len(reasons) > 2 {
			t.Errorf("reasons %v", reasons)
		}
		if cat.gotQ.Group != model.GroupEat || cat.gotQ.Radius != defaultPickRadius {
			t.Errorf("query %+v", cat.gotQ)
		}
	})
	t.Run("limit cuts the list and says more follow", func(t *testing.T) {
		var list api.RecommendationList
		_ = json.Unmarshal(call(h, "GET", base+"&limit=1", "owner", "").Body.Bytes(), &list)
		if len(list.Items) != 1 || !list.HasMore {
			t.Fatalf("items %d hasMore %v", len(list.Items), list.HasMore)
		}
	})
	t.Run("explicit position needs both numbers", func(t *testing.T) {
		if got := call(h, "GET", base+"&lat=21.03", "owner", "").Code; got != 400 {
			t.Errorf("lat only: %d", got)
		}
		if got := call(h, "GET", base+"&lat=21.03&lng=105.85", "owner", "").Code; got != 200 {
			t.Errorf("both: %d", got)
		}
	})
	t.Run("bad input and access", func(t *testing.T) {
		for name, c := range map[string]struct {
			path, token string
			want        int
		}{
			"unknown category": {"/v1/trips/" + trip.Id + "/recommendations?category=spa", "owner", 400},
			"limit too big":    {base + "&limit=99", "owner", 400},
			"non-member":       {base, "other", 404},
			"signed out":       {base, "", 401},
		} {
			if got := call(h, "GET", c.path, c.token, "").Code; got != c.want {
				t.Errorf("%s: got %d, want %d", name, got, c.want)
			}
		}
	})
}

func TestCrewStyleMergesMembers(t *testing.T) {
	trips := newFakeTrips()
	h := &handlers{trips: trips, profile: &fakeProfile{user: model.User{TravelStyle: model.TravelStyle{Interests: []string{"nightlife", "beaches"}, Spend: "splurge"}}}}
	trips.members["t"] = map[string]model.Member{
		"a": {UID: "a", Status: model.MemberActive},
		"b": {UID: "b", Status: model.MemberActive},
		"c": {UID: "c", Status: model.MemberLeft},
	}
	st, err := h.crewStyle(context.Background(), "t")
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Interests) != 2 || st.Interests[0] != "beaches" || st.Spend != 3 {
		t.Errorf("style %+v", st)
	}
}

func TestFreePlanSeesTopFivePicks(t *testing.T) {
	var places []model.NearbyPlace
	for i := 0; i < 8; i++ {
		places = append(places, model.NearbyPlace{CatalogPlace: model.CatalogPlace{
			ID: fmt.Sprintf("p%d", i), Name: map[string]string{"en": "Place"}, Lat: 21.03, Lng: 105.85, Group: model.GroupSee,
		}, Distance: float64(i)})
	}
	for _, tc := range []struct {
		name     string
		plans    fakePlans
		wantN    int
		wantMore bool
	}{
		{"free", fakePlans{}, 5, true},
		{"plus", fakePlans{user: entitlements.User{Plus: true}}, 8, false},
		{"trip pass", fakePlans{trip: entitlements.Trip{Pass: true}}, 8, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"},
				WithTrips(newFakeTrips()), WithStops(&fakeStops{stops: map[string]model.Stop{}}), WithCatalog(&fakeCatalog{places: places}),
				WithEntitlements(entitlements.New(entitlements.Free()), tc.plans))
			var trip api.Trip
			_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
			rec := call(h, "GET", "/v1/trips/"+trip.Id+"/recommendations?category=see&lat=21.03&lng=105.85", "owner", "")
			var list api.RecommendationList
			_ = json.Unmarshal(rec.Body.Bytes(), &list)
			if rec.Code != 200 || len(list.Items) != tc.wantN || list.HasMore != tc.wantMore {
				t.Fatalf("code %d items %d hasMore %v, want %d / %v", rec.Code, len(list.Items), list.HasMore, tc.wantN, tc.wantMore)
			}
		})
	}
}
