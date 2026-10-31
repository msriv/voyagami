package geohash

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

func TestEncodeKnownValue(t *testing.T) {
	// The example from the geohash specification.
	if got := Encode(57.64911, 10.40744, 11); got != "u4pruydqqvj" {
		t.Fatalf("got %s", got)
	}
}

func distanceM(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371000.0
	p1, p2 := lat1*math.Pi/180, lat2*math.Pi/180
	dp, dl := p2-p1, (lng2-lng1)*math.Pi/180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}

func TestCoverHoldsEveryPointInRadius(t *testing.T) {
	rng := rand.New(rand.NewSource(1)) //nolint:gosec // fixed seed keeps the test repeatable
	centres := [][2]float64{{21.0285, 105.8542}, {51.5, -0.12}, {-33.87, 151.2}, {35.68, 139.69}, {0.1, 179.99}}
	for _, c := range centres {
		for _, radius := range []float64{300, 2000, 15000} {
			cover := Cover(c[0], c[1], radius, MaxPrecision)
			if len(cover) == 0 || len(cover) > 9 {
				t.Fatalf("cover size %d", len(cover))
			}
			for i := 0; i < 400; i++ {
				// A random point inside the radius.
				ang, d := rng.Float64()*2*math.Pi, rng.Float64()*radius*0.999
				lat := c[0] + d*math.Cos(ang)/metersPerDegree
				lng := c[1] + d*math.Sin(ang)/(metersPerDegree*math.Cos(c[0]*math.Pi/180))
				if lng > 180 {
					lng -= 360
				}
				if distanceM(c[0], c[1], lat, lng) > radius {
					continue
				}
				h := Encode(lat, lng, MaxPrecision)
				found := false
				for _, p := range cover {
					found = found || strings.HasPrefix(h, p)
				}
				if !found {
					t.Fatalf("centre %v radius %.0f: point %.5f,%.5f (%s) not covered by %v", c, radius, lat, lng, h, cover)
				}
			}
		}
	}
}
