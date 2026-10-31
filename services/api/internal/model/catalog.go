package model

// PlaceGroup is one of our six groups for the places catalogue.
type PlaceGroup string

// The six groups every catalogue place maps to.
const (
	GroupSee       PlaceGroup = "see"
	GroupEat       PlaceGroup = "eat"
	GroupStay      PlaceGroup = "stay"
	GroupDo        PlaceGroup = "do"
	GroupShop      PlaceGroup = "shop"
	GroupNightlife PlaceGroup = "nightlife"
)

// ValidPlaceGroup reports whether g is one of the six groups.
func ValidPlaceGroup(g PlaceGroup) bool {
	switch g {
	case GroupSee, GroupEat, GroupStay, GroupDo, GroupShop, GroupNightlife:
		return true
	}
	return false
}

// PlacePhoto is an openly licensed photo. Its attribution is shown wherever it is.
type PlacePhoto struct {
	URL         string
	Attribution string
	Source      string
	Licence     string
}

// PlaceSource says where one field of a place came from and under which licence.
type PlaceSource struct {
	Source   string
	Licence  string
	SourceID string
}

// CatalogPlace is a place in our own catalogue (collection places), written by
// the places pipeline. It is not Google data.
type CatalogPlace struct {
	ID          string
	Name        map[string]string // by BCP 47 language
	Lat, Lng    float64
	Address     string
	Group       PlaceGroup
	Categories  []string
	CountryCode string
	TimeZone    string
	Photos      []PlacePhoto
	// Provenance is the source and licence per field name.
	Provenance map[string]PlaceSource
	// Ranking inputs, zero when unknown.
	Rating     float64
	Reviews    int
	PriceLevel int
	Tags       []string
}

// DisplayName picks the name in lang, then English, then any name.
func (p CatalogPlace) DisplayName(lang string) string {
	if n := p.Name[lang]; n != "" {
		return n
	}
	if n := p.Name["en"]; n != "" {
		return n
	}
	best := ""
	for l, n := range p.Name {
		if n != "" && (best == "" || l < best) { // smallest language code, so the answer is stable
			best = n
		}
	}
	return best
}

// NearbyQuery asks for catalogue places around a point, nearest first.
type NearbyQuery struct {
	Lat, Lng float64
	Radius   float64 // metres
	Group    PlaceGroup
	Limit    int
	// After skips everything at or before this place in (distance, id) order.
	After *NearbyCursor
}

// NearbyCursor is where the previous page ended.
type NearbyCursor struct {
	Distance float64
	ID       string
}

// NearbyPlace is a catalogue place with its distance from the query point.
type NearbyPlace struct {
	CatalogPlace
	Distance float64 // metres
}
