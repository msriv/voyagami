// Package ranking scores places for "Picks for you". It is a pure function:
// the same inputs always give the same order, and nothing here reads a
// database, a clock or the network.
//
// The signals come from the Recs Spec board: fit to the crew's style, the
// crew's own saves, distance to the stay, opening hours at the planned time,
// quality (rating weighted by review count) and variety. Commission and
// whether a place pays us are never inputs: Candidate has no field for them.
package ranking

import (
	"math"
	"sort"
)

// Weights of each signal in the score. They add up to 1, so a score is 0 to 1.
// The board names the signals but not the numbers; these are our starting
// point and the golden test shows when a change moves the order.
const (
	WeightStyle      = 0.25
	WeightQuality    = 0.20
	WeightDistance   = 0.20
	WeightPopularity = 0.15
	WeightTime       = 0.10
	WeightData       = 0.10
)

const (
	// priorRating and priorWeight pull a rating with few reviews towards the
	// middle, so one 5-star review does not beat 800 reviews at 4.6.
	priorRating = 3.8
	priorWeight = 20.0
	// halfDistanceMeters is how far away a place scores half on distance.
	halfDistanceMeters = 2000.0
	// crewSavesFull is the number of crew saves that counts as fully popular.
	crewSavesFull = 100.0
	// repeatPenalty multiplies the score for each earlier pick with the same kind.
	repeatPenalty = 0.85
)

// Place is what is known about a candidate. Zero values mean unknown.
type Place struct {
	ID   string
	Name string
	// Kind is a fine category such as "temple" or "street food". Variety works on it.
	Kind string
	// Tags are the interests this place suits, in the same words as the crew's
	// style answers (for example "food", "history", "nightlife").
	Tags     []string
	Lat, Lng float64
	HasCoord bool
	// Rating is 0 to 5, 0 when unknown. Reviews is how many ratings it rests on.
	Rating  float64
	Reviews int
	// PriceLevel is 1 (cheap) to 4, 0 when unknown.
	PriceLevel int
	// CrewSaves counts the crews that saved it. MustSee is a curated "must see".
	CrewSaves int
	MustSee   bool
	// Completeness is 0 to 1: how much of name, address, hours, photo and
	// description we have, with the source and licence kept.
	Completeness float64
	// OpensMin and ClosesMin are minutes after midnight on the day in question.
	// Both zero means unknown. ClosesMin below OpensMin means it closes after midnight.
	OpensMin, ClosesMin int
}

// Style is the crew's answers from onboarding, merged.
type Style struct {
	Interests []string
	// Spend is 1 (budget) to 3 (splurge); 0 when unknown.
	Spend int
}

// Context is where and when the pick is for.
type Context struct {
	Style Style
	// Anchor is the stay or the day's stops.
	AnchorLat, AnchorLng float64
	HasAnchor            bool
	// MinuteOfDay is when the person plans to go, or -1 when unknown.
	MinuteOfDay int
}

// Signal names a reason a place scored well.
type Signal string

// The signals, which double as reason codes.
const (
	SignalStyle      Signal = "style"
	SignalQuality    Signal = "quality"
	SignalDistance   Signal = "distance"
	SignalPopularity Signal = "popularity"
	// SignalTime scores opening hours. It is not a reason: closed places are dropped, not explained.
	SignalTime Signal = "time"
)

// Scored is a place with its score and the signals behind it.
type Scored struct {
	Place Place
	Score float64
	// Reasons are the two strongest signals, strongest first. A signal only
	// counts as a reason when it is clearly good and rests on real data.
	Reasons []Signal
	// Parts holds each signal's 0 to 1 value, for tests and tuning.
	Parts map[Signal]float64
	// DistanceMeters is set when both ends have coordinates.
	DistanceMeters float64
}

// CrewMinimum is how many crews must have saved a place before the app may
// say so. Below it the saves still count for ranking but are never shown.
const CrewMinimum = 20

const (
	reasonFloor         = 0.6
	qualityReasonFloor  = 0.8 // about 4.2 stars or better once weighted
	minReviewsForReason = 50
)

// Rank scores every place and returns them best first. Ties break by ID so
// the order never changes between runs. Places that are known to be closed at
// the planned time are dropped.
func Rank(places []Place, ctx Context) []Scored {
	scored := make([]Scored, 0, len(places))
	for _, p := range places {
		if open, known := openAt(p, ctx.MinuteOfDay); known && !open {
			continue
		}
		scored = append(scored, score(p, ctx))
	}
	sortScored(scored)
	return diversify(scored)
}

func score(p Place, ctx Context) Scored {
	s := Scored{Place: p, Parts: map[Signal]float64{}}
	s.Parts[SignalStyle] = styleFit(p, ctx.Style)
	s.Parts[SignalQuality] = quality(p)
	s.Parts[SignalPopularity] = popularity(p)
	s.Parts[SignalTime] = timeFit(p, ctx.MinuteOfDay)
	dist := 0.5 // unknown distance is neither good nor bad
	if ctx.HasAnchor && p.HasCoord {
		s.DistanceMeters = haversine(ctx.AnchorLat, ctx.AnchorLng, p.Lat, p.Lng)
		dist = 1 / (1 + s.DistanceMeters/halfDistanceMeters)
	}
	s.Parts[SignalDistance] = dist
	s.Score = WeightStyle*s.Parts[SignalStyle] +
		WeightQuality*s.Parts[SignalQuality] +
		WeightDistance*dist +
		WeightPopularity*s.Parts[SignalPopularity] +
		WeightTime*s.Parts[SignalTime] +
		WeightData*clamp01(p.Completeness)
	s.Reasons = reasons(s, ctx, p)
	return s
}

// styleFit is the share of the crew's interests the place suits, nudged by
// how well its price matches their spending.
func styleFit(p Place, st Style) float64 {
	if len(st.Interests) == 0 {
		return 0.5
	}
	tags := map[string]bool{}
	for _, t := range p.Tags {
		tags[t] = true
	}
	hit := 0
	for _, in := range st.Interests {
		if tags[in] {
			hit++
		}
	}
	fit := float64(hit) / float64(len(st.Interests))
	if st.Spend > 0 && p.PriceLevel > 0 {
		// Spend 1..3 maps to price 1..4 (1, 2.5, 4).
		want := 1 + float64(st.Spend-1)*1.5
		fit = 0.8*fit + 0.2*(1-math.Min(math.Abs(float64(p.PriceLevel)-want)/3, 1))
	}
	return clamp01(fit)
}

// quality is the rating weighted by how many reviews back it, from 0 to 1.
// No rating gives 0.5, so a new place is not buried.
func quality(p Place) float64 {
	if p.Rating <= 0 {
		return 0.5
	}
	n := float64(p.Reviews)
	r := (p.Rating*n + priorRating*priorWeight) / (n + priorWeight)
	return clamp01((r - 1) / 4)
}

func popularity(p Place) float64 {
	v := math.Log1p(float64(p.CrewSaves)) / math.Log1p(crewSavesFull)
	if p.MustSee {
		v += 0.5
	}
	return clamp01(v)
}

// openAt says whether the place is open at the minute, and whether that is known.
func openAt(p Place, minute int) (open, known bool) {
	if minute < 0 || (p.OpensMin == 0 && p.ClosesMin == 0) {
		return false, false
	}
	if p.ClosesMin == p.OpensMin { // open round the clock
		return true, true
	}
	if p.ClosesMin > p.OpensMin {
		return minute >= p.OpensMin && minute < p.ClosesMin, true
	}
	return minute >= p.OpensMin || minute < p.ClosesMin, true
}

func timeFit(p Place, minute int) float64 {
	open, known := openAt(p, minute)
	switch {
	case !known:
		return 0.5
	case open:
		return 1
	}
	return 0
}

func reasons(s Scored, ctx Context, p Place) []Signal {
	type cand struct {
		sig Signal
		v   float64
	}
	var c []cand
	for _, sig := range []Signal{SignalStyle, SignalDistance, SignalQuality, SignalPopularity} {
		v := s.Parts[sig]
		switch sig {
		case SignalStyle:
			if len(ctx.Style.Interests) == 0 {
				continue
			}
		case SignalDistance:
			if !ctx.HasAnchor || !p.HasCoord {
				continue
			}
		case SignalQuality:
			// The prior lifts every rated place near 0.7, so a reason needs real backing.
			if p.Reviews < minReviewsForReason || v < qualityReasonFloor {
				continue
			}
		case SignalPopularity:
			if p.CrewSaves < CrewMinimum && !p.MustSee {
				continue
			}
		}
		if v >= reasonFloor {
			c = append(c, cand{sig, v})
		}
	}
	sort.SliceStable(c, func(i, j int) bool { return c[i].v > c[j].v })
	var out []Signal
	for i := 0; i < len(c) && i < 2; i++ {
		out = append(out, c[i].sig)
	}
	return out
}

func sortScored(s []Scored) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Score != s[j].Score {
			return s[i].Score > s[j].Score
		}
		return s[i].Place.ID < s[j].Place.ID
	})
}

// diversify picks greedily: each pick's score is cut for every earlier pick
// of the same kind, so five temples in a row become one temple, then a market.
func diversify(in []Scored) []Scored {
	left := append([]Scored(nil), in...)
	out := make([]Scored, 0, len(in))
	seen := map[string]int{}
	for len(left) > 0 {
		best, bestEff := 0, -1.0
		for i, s := range left {
			eff := s.Score * math.Pow(repeatPenalty, float64(seen[s.Place.Kind]))
			// left is sorted by raw score, so on a tie the earlier one (lower ID) wins.
			if eff > bestEff {
				best, bestEff = i, eff
			}
		}
		pick := left[best]
		if pick.Place.Kind != "" {
			seen[pick.Place.Kind]++
		}
		out = append(out, pick)
		left = append(left[:best], left[best+1:]...)
	}
	return out
}

func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const earth = 6371000.0
	rad := math.Pi / 180
	dLat, dLng := (lat2-lat1)*rad, (lng2-lng1)*rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earth * math.Asin(math.Min(1, math.Sqrt(a)))
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
