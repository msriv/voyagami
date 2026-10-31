// Package geohash encodes map positions as geohash strings, so a "places near
// here" query becomes a few prefix ranges on one indexed field.
package geohash

import (
	"math"
	"strings"
)

const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"

// MaxPrecision is the length stored on every place (about 5 m cells).
const MaxPrecision = 9

// Encode returns the geohash of a position with the given number of characters.
func Encode(lat, lng float64, precision int) string {
	latR, lngR := [2]float64{-90, 90}, [2]float64{-180, 180}
	var sb strings.Builder
	bit, ch, even := 0, 0, true
	for sb.Len() < precision {
		r, v := &latR, lat
		if even {
			r, v = &lngR, lng
		}
		mid := (r[0] + r[1]) / 2
		ch <<= 1
		if v >= mid {
			ch |= 1
			r[0] = mid
		} else {
			r[1] = mid
		}
		even = !even
		if bit++; bit == 5 {
			sb.WriteByte(base32[ch])
			bit, ch = 0, 0
		}
	}
	return sb.String()
}

// cellSize is the height and width in degrees of a cell with this many characters.
func cellSize(precision int) (dLat, dLng float64) {
	bits := 5 * precision
	return 180 / math.Pow(2, float64(bits/2)), 360 / math.Pow(2, float64((bits+1)/2))
}

const metersPerDegree = 111320.0

// Cover returns the geohash prefixes whose cells together hold every point
// within radiusMeters of the position. It is the cell of the position and its
// eight neighbours, at the longest precision whose cells are still wider than
// the radius.
//
// shortcut: near the poles a cell is narrower than the radius in metres, so a
// few points can be missed there; fine for the pilot city, revisit with polar regions.
func Cover(lat, lng, radiusMeters float64, maxPrecision int) []string {
	p := 1
	for q := 2; q <= maxPrecision; q++ {
		dLat, dLng := cellSize(q)
		widthM := dLng * metersPerDegree * math.Max(math.Cos(lat*math.Pi/180), 0.01)
		if dLat*metersPerDegree < radiusMeters || widthM < radiusMeters {
			break
		}
		p = q
	}
	dLat, dLng := cellSize(p)
	seen := map[string]bool{}
	var out []string
	for _, a := range []float64{-1, 0, 1} {
		for _, b := range []float64{-1, 0, 1} {
			la := math.Max(-90, math.Min(90, lat+a*dLat))
			ln := lng + b*dLng
			if ln > 180 {
				ln -= 360
			} else if ln < -180 {
				ln += 360
			}
			h := Encode(la, ln, p)
			if !seen[h] {
				seen[h] = true
				out = append(out, h)
			}
		}
	}
	return out
}
