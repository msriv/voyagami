package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/stays"
)

type expClicks struct{ got []string }

func (e *expClicks) Add(_ context.Context, id string, _ time.Time) error {
	e.got = append(e.got, id)
	return nil
}

func TestExperiences(t *testing.T) {
	cat := &fakeCatalog{places: []model.NearbyPlace{
		{CatalogPlace: model.CatalogPlace{ID: "p1", Name: map[string]string{"en": "Water puppets"}, Group: model.GroupDo}, Distance: 120.4},
	}}
	partners := []stays.Partner{{ID: "klook", Name: "Klook", BaseURL: "https://klook.example/search", Params: stays.Params{Destination: "q"}, Fixed: map[string]string{"aid": "42"}}}
	clicks := &expClicks{}
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner", "other": "u-other"},
		WithTrips(newFakeTrips()), WithStops(&fakeStops{stops: map[string]model.Stop{}}), WithCatalog(cat), WithExperiences(partners, clicks))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	path := "/v1/trips/" + trip.Id + "/experiences"

	var l api.ExperienceList
	rec := call(h, "GET", path, "owner", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &l)
	if rec.Code != 200 || len(l.Items) != 0 {
		t.Fatalf("no stops: %d %s", rec.Code, rec.Body)
	}

	call(h, "POST", "/v1/trips/"+trip.Id+"/stops", "owner", `{"date":"2027-02-02","place":{"name":"Hotel","lat":21.03,"lng":105.85}}`)
	rec = call(h, "GET", path, "owner", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &l)
	if rec.Code != 200 || len(l.Items) != 1 || l.Items[0].PlaceId != "p1" || l.Items[0].DistanceMeters != 120 ||
		len(l.Items[0].Links) != 1 || !strings.Contains(l.Items[0].Links[0].Url, "aid=42") || !strings.Contains(l.Items[0].Links[0].Url, "q=Water+puppets") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if cat.gotQ.Group != model.GroupDo {
		t.Errorf("asked for %q, want the do group", cat.gotQ.Group)
	}

	if c := call(h, "POST", path+"/clicks", "owner", `{"partnerId":"klook"}`).Code; c != 204 || len(clicks.got) != 1 || clicks.got[0] != "exp-klook" {
		t.Errorf("click: %d %v", c, clicks.got)
	}
	for name, c := range map[string]struct {
		method, p, token, body string
		want                   int
	}{
		"unknown partner": {"POST", path + "/clicks", "owner", `{"partnerId":"nope"}`, 400},
		"non-member":      {"GET", path, "other", "", 404},
		"signed out":      {"GET", path, "", "", 401},
	} {
		if got := call(h, c.method, c.p, c.token, c.body).Code; got != c.want {
			t.Errorf("%s: %d, want %d", name, got, c.want)
		}
	}
}

func TestExperiencesOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	if c := call(h, "GET", "/v1/trips/t/experiences", "owner", "").Code; c != 503 {
		t.Errorf("%d", c)
	}
}
