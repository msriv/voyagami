package places

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ErrNoPlace means the link did not point at a place we can read.
var ErrNoPlace = errors.New("places: no place in that link")

var (
	atPin    = regexp.MustCompile(`@(-?\d{1,2}(?:\.\d+)?),(-?\d{1,3}(?:\.\d+)?)`)
	dataPin  = regexp.MustCompile(`!3d(-?\d{1,2}(?:\.\d+)?)!4d(-?\d{1,3}(?:\.\d+)?)`)
	pairPin  = regexp.MustCompile(`^\s*(-?\d{1,2}(?:\.\d+)?)\s*,\s*(-?\d{1,3}(?:\.\d+)?)\s*$`)
	osmHash  = regexp.MustCompile(`map=\d+/(-?\d{1,2}(?:\.\d+)?)/(-?\d{1,3}(?:\.\d+)?)`)
	placeSeg = regexp.MustCompile(`/place/([^/]+)`)
)

// shortHosts redirect to a full maps link; we follow them (headers only,
// never reading a body) but never leave this list.
var shortHosts = map[string]bool{"maps.app.goo.gl": true, "goo.gl": true, "g.co": true}

func isMapsHost(h string) bool {
	h = strings.TrimPrefix(strings.ToLower(h), "www.")
	return h == "google.com" || h == "maps.google.com" || strings.HasSuffix(h, ".google.com") || strings.HasPrefix(h, "google.") ||
		h == "openstreetmap.org" || h == "osm.org" || shortHosts[h]
}

// Resolve turns a pasted maps link into a place candidate. The link is data:
// only an allow-listed maps host is ever contacted, and only to follow a
// short link's redirect.
func Resolve(ctx context.Context, raw string) (Result, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return Result{}, ErrNoPlace
	}
	if u.Scheme == "geo" {
		r := pinFrom(u.Opaque, "")
		return r, okOrNone(r)
	}
	if (u.Scheme != "https" && u.Scheme != "http") || !isMapsHost(u.Hostname()) {
		return Result{}, ErrNoPlace
	}
	if shortHosts[strings.ToLower(u.Hostname())] {
		if u, err = follow(ctx, u); err != nil {
			return Result{}, ErrNoPlace
		}
	}
	r := fromURL(u)
	return r, okOrNone(r)
}

func okOrNone(r Result) error {
	if r.Lat == 0 && r.Lng == 0 {
		return ErrNoPlace
	}
	return nil
}

func pinFrom(s, name string) Result {
	m := pairPin.FindStringSubmatch(strings.SplitN(s, ";", 2)[0])
	if m == nil {
		return Result{}
	}
	return mk(m[1], m[2], name)
}

func mk(lat, lng, name string) Result {
	a, _ := strconv.ParseFloat(lat, 64)
	b, _ := strconv.ParseFloat(lng, 64)
	if a < -90 || a > 90 || b < -180 || b > 180 {
		return Result{}
	}
	return Result{Name: name, Lat: a, Lng: b}
}

func fromURL(u *url.URL) Result {
	name := ""
	if m := placeSeg.FindStringSubmatch(u.EscapedPath()); m != nil {
		if n, err := url.PathUnescape(m[1]); err == nil {
			name = strings.TrimSpace(strings.ReplaceAll(n, "+", " "))
		}
	}
	full := u.EscapedPath() + "?" + u.RawQuery + "#" + u.Fragment
	if m := dataPin.FindStringSubmatch(full); m != nil {
		return mk(m[1], m[2], name)
	}
	if m := atPin.FindStringSubmatch(full); m != nil {
		return mk(m[1], m[2], name)
	}
	if m := osmHash.FindStringSubmatch(full); m != nil {
		return mk(m[1], m[2], name)
	}
	q := u.Query()
	for _, k := range []string{"q", "query", "ll", "center"} {
		if r := pinFrom(q.Get(k), name); r.Lat != 0 || r.Lng != 0 {
			return r
		}
	}
	if q.Get("mlat") != "" {
		return mk(q.Get("mlat"), q.Get("mlon"), name)
	}
	return Result{}
}

func follow(ctx context.Context, u *url.URL) (*url.URL, error) {
	c := &http.Client{
		Timeout: httpClient.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 4 || !isMapsHost(req.URL.Hostname()) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	_ = res.Body.Close()
	if !isMapsHost(res.Request.URL.Hostname()) {
		return nil, ErrNoPlace
	}
	return res.Request.URL, nil
}
