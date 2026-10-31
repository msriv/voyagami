package firestorerepo

import (
	"context"
	"errors"
	"math"
	"sort"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/msriv/voyagami/services/api/internal/geohash"
	"github.com/msriv/voyagami/services/api/internal/model"
	"github.com/msriv/voyagami/services/api/internal/repo"
)

const placesCollection = "places"

// perCellCap bounds how many documents one geohash cell may return.
//
// shortcut: a dense cell beyond this many places loses the farthest ones; the
// pilot city is well below it. Move to a search index when cities grow (P1).
const perCellCap = 1000

// Catalog implements repo.PlaceCatalog on the places collection.
type Catalog struct {
	client *firestore.Client
}

var _ repo.PlaceCatalog = (*Catalog)(nil)

// NewCatalog returns the Firestore-backed places catalogue reader.
func NewCatalog(client *firestore.Client) *Catalog { return &Catalog{client: client} }

type catalogDoc struct {
	Name     map[string]string `firestore:"name"`
	Location struct {
		Lat float64 `firestore:"lat"`
		Lng float64 `firestore:"lng"`
	} `firestore:"location"`
	Geohash string `firestore:"geohash"`
	Address struct {
		Formatted string `firestore:"formatted"`
	} `firestore:"address"`
	Group       string   `firestore:"group"`
	Categories  []string `firestore:"categories"`
	CountryCode string   `firestore:"countryCode"`
	TimeZone    string   `firestore:"timeZone"`
	Photos      []struct {
		URL         string `firestore:"url"`
		Attribution string `firestore:"attribution"`
		Source      string `firestore:"source"`
		Licence     string `firestore:"licence"`
	} `firestore:"photos"`
	Provenance map[string]struct {
		Source   string `firestore:"source"`
		Licence  string `firestore:"licence"`
		SourceID string `firestore:"sourceId"`
	} `firestore:"provenance"`
	Rating     float64  `firestore:"rating"`
	Reviews    int      `firestore:"reviews"`
	PriceLevel int      `firestore:"priceLevel"`
	Tags       []string `firestore:"tags"`
}

func (d catalogDoc) place(id string) model.CatalogPlace {
	p := model.CatalogPlace{
		ID: id, Name: d.Name, Lat: d.Location.Lat, Lng: d.Location.Lng, Address: d.Address.Formatted,
		Group: model.PlaceGroup(d.Group), Categories: d.Categories, CountryCode: d.CountryCode, TimeZone: d.TimeZone,
		Rating: d.Rating, Reviews: d.Reviews, PriceLevel: d.PriceLevel, Tags: d.Tags,
	}
	for _, ph := range d.Photos {
		p.Photos = append(p.Photos, model.PlacePhoto{URL: ph.URL, Attribution: ph.Attribution, Source: ph.Source, Licence: ph.Licence})
	}
	if len(d.Provenance) > 0 {
		p.Provenance = map[string]model.PlaceSource{}
		for f, s := range d.Provenance {
			p.Provenance[f] = model.PlaceSource{Source: s.Source, Licence: s.Licence, SourceID: s.SourceID}
		}
	}
	return p
}

// Get returns one place.
func (c *Catalog) Get(ctx context.Context, id string) (model.CatalogPlace, error) {
	if id == "" {
		return model.CatalogPlace{}, repo.ErrNotFound
	}
	snap, err := c.client.Collection(placesCollection).Doc(id).Get(ctx)
	if err != nil {
		return model.CatalogPlace{}, mapErr("get place", err)
	}
	var d catalogDoc
	if err := snap.DataTo(&d); err != nil {
		return model.CatalogPlace{}, mapErr("decode place", err)
	}
	return d.place(snap.Ref.ID), nil
}

// Nearby reads the geohash cells that cover the circle, keeps the places inside
// it and pages by (distance, id).
func (c *Catalog) Nearby(ctx context.Context, q model.NearbyQuery) ([]model.NearbyPlace, bool, error) {
	var found []model.NearbyPlace
	for _, prefix := range geohash.Cover(q.Lat, q.Lng, q.Radius, geohash.MaxPrecision) {
		query := c.client.Collection(placesCollection).Where("geohash", ">=", prefix).Where("geohash", "<", prefix+"{")
		if q.Group != "" {
			query = query.Where("group", "==", string(q.Group))
		}
		it := query.Limit(perCellCap).Documents(ctx)
		for {
			snap, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				it.Stop()
				return nil, false, mapErr("list nearby places", err)
			}
			var d catalogDoc
			if err := snap.DataTo(&d); err != nil {
				it.Stop()
				return nil, false, mapErr("decode place", err)
			}
			dist := distanceMeters(q.Lat, q.Lng, d.Location.Lat, d.Location.Lng)
			if dist > q.Radius {
				continue
			}
			found = append(found, model.NearbyPlace{CatalogPlace: d.place(snap.Ref.ID), Distance: dist})
		}
		it.Stop()
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].Distance != found[j].Distance {
			return found[i].Distance < found[j].Distance
		}
		return found[i].ID < found[j].ID
	})
	if q.After != nil {
		i := sort.Search(len(found), func(i int) bool {
			if found[i].Distance != q.After.Distance {
				return found[i].Distance > q.After.Distance
			}
			return found[i].ID > q.After.ID
		})
		found = found[i:]
	}
	if q.Limit > 0 && len(found) > q.Limit {
		return found[:q.Limit], true, nil
	}
	return found, false, nil
}

// distanceMeters is the great-circle distance between two positions.
func distanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earth = 6371000.0
	p1, p2 := lat1*math.Pi/180, lat2*math.Pi/180
	dp, dl := p2-p1, (lng2-lng1)*math.Pi/180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * earth * math.Asin(math.Sqrt(a))
}
