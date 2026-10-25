package bestorder

import (
	"context"
	"errors"
	"math"
)

// Point is a place on the map.
type Point struct{ Lat, Lng float64 }

// Straight estimates travel time from the straight-line distance at a steady
// speed. It is the stand-in until the directions service is wired in (VOY-128).
type Straight struct {
	// Points are the stop coordinates keyed by stop ID.
	Points map[string]Point
	// KmPerHour is the assumed speed. It must be above zero.
	KmPerHour float64
}

var _ Distances = Straight{}

const earthRadiusKm = 6371.0088

// Matrix implements Distances.
func (s Straight) Matrix(_ context.Context, ids []string) ([][]float64, error) {
	if s.KmPerHour <= 0 {
		return nil, errors.New("bestorder: speed must be above zero")
	}
	m := make([][]float64, len(ids))
	for i, a := range ids {
		pa, ok := s.Points[a]
		if !ok {
			return nil, errors.New("bestorder: no coordinates for stop " + a)
		}
		m[i] = make([]float64, len(ids))
		for j, b := range ids {
			pb, ok := s.Points[b]
			if !ok {
				return nil, errors.New("bestorder: no coordinates for stop " + b)
			}
			m[i][j] = haversineKm(pa, pb) / s.KmPerHour * 60
		}
	}
	return m, nil
}

func haversineKm(a, b Point) float64 {
	rad := math.Pi / 180
	dLat, dLng := (b.Lat-a.Lat)*rad, (b.Lng-a.Lng)*rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(a.Lat*rad)*math.Cos(b.Lat*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

// DistanceKm is the straight-line distance between two points in kilometres.
func DistanceKm(a, b Point) float64 { return haversineKm(a, b) }
