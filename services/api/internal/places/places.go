// Package places searches for places by name. The provider key stays on the
// server; results are shown live and only the pin the person picks is stored.
package places

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Result is one match.
type Result struct {
	Name     string
	Address  string
	Lat, Lng float64
	Category string // one of the stop categories, or empty
}

// Near biases results to a point. Nil means no bias.
type Near struct{ Lat, Lng float64 }

// Searcher looks up places by text.
type Searcher interface {
	Search(ctx context.Context, q string, near *Near) ([]Result, error)
}

const maxResults = 10

var httpClient = &http.Client{Timeout: 8 * time.Second}

// Google uses Places API (New) text search. Needs a key from VOY-7.
type Google struct {
	Key string
	URL string // override in tests
}

// Search calls Google text search and keeps only what the stop needs.
func (g Google) Search(ctx context.Context, q string, near *Near) ([]Result, error) {
	u := g.URL
	if u == "" {
		u = "https://places.googleapis.com/v1/places:searchText"
	}
	body := map[string]any{"textQuery": q, "pageSize": maxResults}
	if near != nil {
		body["locationBias"] = map[string]any{"circle": map[string]any{
			"center": map[string]float64{"latitude": near.Lat, "longitude": near.Lng}, "radius": 50000.0}}
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", g.Key)
	req.Header.Set("X-Goog-FieldMask", "places.displayName,places.formattedAddress,places.location,places.primaryType")
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("places: google answered %d", res.StatusCode)
	}
	var out struct {
		Places []struct {
			DisplayName struct{ Text string } `json:"displayName"`
			Address     string                `json:"formattedAddress"`
			Location    struct{ Latitude, Longitude float64 }
			PrimaryType string `json:"primaryType"`
		} `json:"places"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	var rs []Result
	for _, p := range out.Places {
		if p.DisplayName.Text == "" {
			continue
		}
		rs = append(rs, Result{Name: p.DisplayName.Text, Address: p.Address, Lat: p.Location.Latitude, Lng: p.Location.Longitude, Category: category(p.PrimaryType)})
	}
	return rs, nil
}

func category(t string) string {
	switch {
	case strings.Contains(t, "restaurant"), strings.Contains(t, "cafe"), strings.Contains(t, "bar"), strings.Contains(t, "food"):
		return "food"
	case strings.Contains(t, "hotel"), strings.Contains(t, "lodging"), strings.Contains(t, "hostel"):
		return "stay"
	case strings.Contains(t, "museum"), strings.Contains(t, "tourist"), strings.Contains(t, "temple"), strings.Contains(t, "park"):
		return "sights"
	case strings.Contains(t, "store"), strings.Contains(t, "shopping"), strings.Contains(t, "market"):
		return "shopping"
	case strings.Contains(t, "station"), strings.Contains(t, "airport"):
		return "transport"
	}
	return ""
}

// OSM is the dev fallback while there is no Google key: OpenStreetMap
// Nominatim. shortcut: public instance allows about one request a second and
// needs a contact in the user agent; swap for the pipeline in D4.
type OSM struct {
	URL string // override in tests
}

// Search calls Nominatim.
func (o OSM) Search(ctx context.Context, q string, near *Near) ([]Result, error) {
	u := o.URL
	if u == "" {
		u = "https://nominatim.openstreetmap.org/search"
	}
	v := url.Values{"q": {q}, "format": {"jsonv2"}, "limit": {strconv.Itoa(maxResults)}}
	if near != nil {
		v.Set("viewbox", fmt.Sprintf("%f,%f,%f,%f", near.Lng-0.5, near.Lat+0.5, near.Lng+0.5, near.Lat-0.5))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "voyagami-dev/0.1 (https://github.com/msriv/voyagami)")
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("places: nominatim answered %d", res.StatusCode)
	}
	var rows []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
		return nil, err
	}
	var rs []Result
	for _, r := range rows {
		lat, _ := strconv.ParseFloat(str(r["lat"]), 64)
		lng, _ := strconv.ParseFloat(str(r["lon"]), 64)
		name := str(r["name"])
		addr := str(r["display_name"])
		if name == "" {
			name, _, _ = strings.Cut(addr, ",")
		}
		rs = append(rs, Result{Name: name, Address: addr, Lat: lat, Lng: lng, Category: category(str(r["type"]))})
	}
	return rs, nil
}

func str(v any) string { s, _ := v.(string); return s }
