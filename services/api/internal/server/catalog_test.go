package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

type fakeCatalog struct {
	places []model.NearbyPlace
	gotQ   model.NearbyQuery
}

func (f *fakeCatalog) Get(_ context.Context, id string) (model.CatalogPlace, error) {
	for _, p := range f.places {
		if p.ID == id {
			return p.CatalogPlace, nil
		}
	}
	return model.CatalogPlace{}, repo.ErrNotFound
}

func (f *fakeCatalog) Nearby(_ context.Context, q model.NearbyQuery) ([]model.NearbyPlace, bool, error) {
	f.gotQ = q
	if len(f.places) > q.Limit {
		return f.places[:q.Limit], true, nil
	}
	return f.places, false, nil
}

func catalogRig() (*fakeCatalog, func(method, path, token string) (int, []byte)) {
	lake := model.CatalogPlace{
		ID: "p1", Name: map[string]string{"vi": "Hồ Hoàn Kiếm", "en": "Hoan Kiem Lake"}, Lat: 21.0285, Lng: 105.852,
		Group: model.GroupSee, Categories: []string{"park"}, CountryCode: "VN",
		Photos:     []model.PlacePhoto{{URL: "https://img.example/lake.jpg", Attribution: "Photo by A, CC BY-SA 4.0", Source: "wikimedia", Licence: "CC-BY-SA-4.0"}},
		Provenance: map[string]model.PlaceSource{"name": {Source: "osm", Licence: "ODbL"}, "location": {Source: "foursquare", Licence: "Apache-2.0", SourceID: "fsq1"}},
	}
	pho := model.CatalogPlace{ID: "p2", Name: map[string]string{"en": "Pho 10"}, Lat: 21.03, Lng: 105.85, Group: model.GroupEat}
	f := &fakeCatalog{places: []model.NearbyPlace{{CatalogPlace: lake, Distance: 120.5}, {CatalogPlace: pho, Distance: 800}}}
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithCatalog(f))
	return f, func(method, path, token string) (int, []byte) {
		rec := call(h, method, path, token, "")
		return rec.Code, rec.Body.Bytes()
	}
}

func TestNearbyPlaces(t *testing.T) {
	f, get := catalogRig()
	code, body := get("GET", "/v1/places/nearby?lat=21.03&lng=105.85&limit=1&lang=vi&group=see", "owner")
	if code != 200 {
		t.Fatalf("%d %s", code, body)
	}
	var list api.CatalogPlaceList
	_ = json.Unmarshal(body, &list)
	if len(list.Places) != 1 || list.Places[0].Name != "Hồ Hoàn Kiếm" || list.NextCursor == nil {
		t.Fatalf("list %+v", list)
	}
	if list.Places[0].DistanceMeters == nil || *list.Places[0].DistanceMeters != 120.5 {
		t.Errorf("distance %v", list.Places[0].DistanceMeters)
	}
	if f.gotQ.Radius != 2000 || f.gotQ.Group != model.GroupSee || f.gotQ.Limit != 1 {
		t.Errorf("query %+v", f.gotQ)
	}
	// The cursor round-trips into the next query.
	if code, _ := get("GET", "/v1/places/nearby?lat=21.03&lng=105.85&cursor="+url.QueryEscape(*list.NextCursor), "owner"); code != 200 {
		t.Fatalf("next page %d", code)
	}
	if f.gotQ.After == nil || f.gotQ.After.ID != "p1" || f.gotQ.After.Distance != 120.5 {
		t.Errorf("after %+v", f.gotQ.After)
	}
}

func TestNearbyPlacesRejectsBadInput(t *testing.T) {
	_, get := catalogRig()
	for name, path := range map[string]string{
		"lat out of range": "/v1/places/nearby?lat=95&lng=105",
		"radius too big":   "/v1/places/nearby?lat=21&lng=105&radiusMeters=50000",
		"limit too big":    "/v1/places/nearby?lat=21&lng=105&limit=500",
		"unknown group":    "/v1/places/nearby?lat=21&lng=105&group=spa",
		"bad cursor":       "/v1/places/nearby?lat=21&lng=105&cursor=!!!",
		"missing lat":      "/v1/places/nearby?lng=105",
	} {
		if code, _ := get("GET", path, "owner"); code != 400 {
			t.Errorf("%s: got %d, want 400", name, code)
		}
	}
	if code, _ := get("GET", "/v1/places/nearby?lat=21&lng=105", ""); code != 401 {
		t.Errorf("signed out: %d", code)
	}
}

func TestGetCatalogPlace(t *testing.T) {
	_, get := catalogRig()
	code, body := get("GET", "/v1/places/p1", "owner")
	if code != 200 {
		t.Fatalf("%d %s", code, body)
	}
	var p api.CatalogPlace
	_ = json.Unmarshal(body, &p)
	if p.Name != "Hoan Kiem Lake" || len(p.Photos) != 1 || p.Photos[0].Attribution == "" {
		t.Fatalf("place %+v", p)
	}
	if len(p.Sources) != 2 || p.Sources[0].Field != "location" || p.Sources[0].SourceId == nil || p.Sources[1].Licence != "ODbL" {
		t.Errorf("sources %+v", p.Sources)
	}
	if code, _ := get("GET", "/v1/places/nope", "owner"); code != 404 {
		t.Errorf("missing: %d", code)
	}
}

func TestCatalogOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	if got := call(h, "GET", "/v1/places/p1", "owner", "").Code; got != 503 {
		t.Fatalf("got %d, want 503", got)
	}
}
