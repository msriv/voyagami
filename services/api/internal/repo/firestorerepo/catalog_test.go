package firestorerepo

import (
	"errors"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/geohash"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

func TestCatalogNearbyAndGet(t *testing.T) {
	c := testClient(t)
	ctx := testCtx(t)
	s := NewCatalog(c)
	// A unique centre far from other tests' data: the middle of the Pacific.
	const lat, lng = -0.5, -150.5
	put := func(id string, dLat, dLng float64, group string) {
		t.Helper()
		doc := map[string]any{
			"name":     map[string]any{"en": id},
			"location": map[string]any{"lat": lat + dLat, "lng": lng + dLng},
			"geohash":  geohash.Encode(lat+dLat, lng+dLng, geohash.MaxPrecision),
			"group":    group, "categories": []string{"x"},
			"photos":     []map[string]any{{"url": "https://img.example/a.jpg", "attribution": "A", "source": "wikimedia", "licence": "CC0"}},
			"provenance": map[string]any{"name": map[string]any{"source": "osm", "licence": "ODbL"}},
		}
		if _, err := c.Collection(placesCollection).Doc(id).Set(ctx, doc); err != nil {
			t.Fatal(err)
		}
	}
	// Distances grow with the offsets (about 111 m per 0.001 degree).
	put("c-a", 0.001, 0, "eat")
	put("c-b", 0.002, 0, "see")
	put("c-c", 0, 0.003, "eat")
	put("c-d", 0.004, 0, "eat")
	put("c-far", 0.5, 0, "eat") // 55 km away

	got, more, err := s.Nearby(ctx, model.NearbyQuery{Lat: lat, Lng: lng, Radius: 1000, Limit: 10})
	if err != nil || more {
		t.Fatalf("nearby: %v more=%v", err, more)
	}
	ids := func(ps []model.NearbyPlace) []string {
		var out []string
		for _, p := range ps {
			out = append(out, p.ID)
		}
		return out
	}
	if want := []string{"c-a", "c-b", "c-c", "c-d"}; !equal(ids(got), want) {
		t.Fatalf("order %v, want %v", ids(got), want)
	}
	if got[0].Distance < 100 || got[0].Distance > 125 {
		t.Errorf("distance %f", got[0].Distance)
	}

	// Radius cuts, group filters.
	if got, _, _ := s.Nearby(ctx, model.NearbyQuery{Lat: lat, Lng: lng, Radius: 250, Limit: 10}); !equal(ids(got), []string{"c-a", "c-b"}) {
		t.Errorf("radius 250: %v", ids(got))
	}
	if got, _, _ := s.Nearby(ctx, model.NearbyQuery{Lat: lat, Lng: lng, Radius: 1000, Limit: 10, Group: model.GroupEat}); !equal(ids(got), []string{"c-a", "c-c", "c-d"}) {
		t.Errorf("group eat: %v", ids(got))
	}

	// Paging walks the list without repeats.
	first, more, _ := s.Nearby(ctx, model.NearbyQuery{Lat: lat, Lng: lng, Radius: 1000, Limit: 2})
	if !more || !equal(ids(first), []string{"c-a", "c-b"}) {
		t.Fatalf("page 1: %v more=%v", ids(first), more)
	}
	last := first[1]
	second, more, _ := s.Nearby(ctx, model.NearbyQuery{Lat: lat, Lng: lng, Radius: 1000, Limit: 2, After: &model.NearbyCursor{Distance: last.Distance, ID: last.ID}})
	if more || !equal(ids(second), []string{"c-c", "c-d"}) {
		t.Fatalf("page 2: %v more=%v", ids(second), more)
	}

	// Detail carries photos and sources.
	p, err := s.Get(ctx, "c-a")
	if err != nil || len(p.Photos) != 1 || p.Photos[0].Licence != "CC0" || p.Provenance["name"].Licence != "ODbL" || p.Name["en"] != "c-a" {
		t.Fatalf("get: %+v %v", p, err)
	}
	if _, err := s.Get(ctx, "missing"); !errors.Is(err, repo.ErrNotFound) {
		t.Errorf("missing: %v", err)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
