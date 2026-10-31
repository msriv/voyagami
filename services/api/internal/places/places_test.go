package places

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Goog-Api-Key") != "k" {
			w.WriteHeader(403)
			return
		}
		_, _ = w.Write([]byte(`{"places":[{"displayName":{"text":"Pho 10"},"formattedAddress":"10 Ly Quoc Su","location":{"latitude":21.03,"longitude":105.85},"primaryType":"vietnamese_restaurant"},{"displayName":{"text":""}}]}`))
	}))
	defer srv.Close()
	rs, err := Google{Key: "k", URL: srv.URL}.Search(context.Background(), "pho", &Near{Lat: 21, Lng: 105})
	if err != nil || len(rs) != 1 || rs[0].Name != "Pho 10" || rs[0].Category != "food" || rs[0].Lat != 21.03 {
		t.Fatalf("got %v %v", rs, err)
	}
	if _, err := (Google{Key: "bad", URL: srv.URL}).Search(context.Background(), "pho", nil); err == nil {
		t.Error("want error on 403")
	}
}

func TestOSMSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"","display_name":"Hoan Kiem Lake, Hanoi","lat":"21.0285","lon":"105.8542","type":"park"}]`))
	}))
	defer srv.Close()
	rs, err := OSM{URL: srv.URL}.Search(context.Background(), "lake", nil)
	if err != nil || len(rs) != 1 || rs[0].Name != "Hoan Kiem Lake" || rs[0].Category != "sights" || rs[0].Lng != 105.8542 {
		t.Fatalf("got %v %v", rs, err)
	}
}
