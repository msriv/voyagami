package places

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestResolveParsesMapsLinks(t *testing.T) {
	cases := []struct {
		link, name string
		lat, lng   float64
	}{
		{"https://www.google.com/maps/place/Hoan+Kiem+Lake/@21.0285,105.8542,15z/data=!3m1", "Hoan Kiem Lake", 21.0285, 105.8542},
		{"https://www.google.com/maps/place/X/data=!3d21.01!4d105.82", "X", 21.01, 105.82},
		{"https://maps.google.com/?q=21.03,105.85", "", 21.03, 105.85},
		{"https://www.openstreetmap.org/#map=17/21.0285/105.8542", "", 21.0285, 105.8542},
		{"https://www.openstreetmap.org/?mlat=21.1&mlon=105.2", "", 21.1, 105.2},
		{"geo:21.03,105.85", "", 21.03, 105.85},
	}
	for _, c := range cases {
		r, err := Resolve(context.Background(), c.link)
		if err != nil || r.Lat != c.lat || r.Lng != c.lng || r.Name != c.name {
			t.Errorf("%s: got %+v %v", c.link, r, err)
		}
	}
}

func TestResolveRefusesOtherLinks(t *testing.T) {
	for _, l := range []string{"", "not a url", "https://evil.example/@21,105", "http://169.254.169.254/", "file:///etc/passwd", "https://www.google.com/maps/@95,105,10z", "https://www.google.com/"} {
		if _, err := Resolve(context.Background(), l); err == nil {
			t.Errorf("%q: want error", l)
		}
	}
}

func TestFollowStaysOnMapsHosts(t *testing.T) {
	away := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("left maps hosts") }))
	defer away.Close()
	short := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, away.URL, http.StatusFound)
	}))
	defer short.Close()
	u := short.URL
	pu, _ := url.Parse(u)
	if _, err := follow(context.Background(), pu); err == nil {
		t.Error("want error when the redirect leaves maps hosts")
	}
}
