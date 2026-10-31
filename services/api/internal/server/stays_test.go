package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/stays"
)

type fakeClicks map[string]int

func (f fakeClicks) Add(_ context.Context, partnerID string, _ time.Time) error {
	f[partnerID]++
	return nil
}

func TestStayRoutes(t *testing.T) {
	partners, err := stays.Load(`[{"id":"alpha","name":"Alpha Stays","baseUrl":"https://alpha.example/s",
	  "params":{"destination":"q","guests":"adults"},"fixed":{"aid":"PLACEHOLDER"}}]`)
	if err != nil {
		t.Fatal(err)
	}
	clicks := fakeClicks{}
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner", "other": "u-other"},
		WithTrips(newFakeTrips()), WithStays(partners, clicks))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)

	rec := call(h, "GET", "/v1/trips/"+trip.Id+"/stays/links?guests=3", "owner", "")
	if rec.Code != 200 {
		t.Fatalf("links: %d %s", rec.Code, rec.Body)
	}
	var list api.StayLinkList
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Links) != 1 || list.Links[0].PartnerId != "alpha" {
		t.Fatalf("links: %+v", list)
	}
	u, _ := url.Parse(list.Links[0].Url)
	if u.Query().Get("adults") != "3" || u.Query().Get("aid") != "PLACEHOLDER" || u.Query().Get("q") == "" {
		t.Errorf("url %s", list.Links[0].Url)
	}

	for _, c := range []struct {
		name, method, path, token, body string
		want                            int
	}{
		{"click", "POST", "/stays/clicks", "owner", `{"partnerId":"alpha"}`, 204},
		{"unknown partner", "POST", "/stays/clicks", "owner", `{"partnerId":"nope"}`, 400},
		{"non-member click", "POST", "/stays/clicks", "other", `{"partnerId":"alpha"}`, 404},
		{"non-member links", "GET", "/stays/links", "other", "", 404},
		{"signed out", "GET", "/stays/links", "", "", 401},
	} {
		if got := call(h, c.method, "/v1/trips/"+trip.Id+c.path, c.token, c.body).Code; got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
	if clicks["alpha"] != 1 || len(clicks) != 1 {
		t.Errorf("clicks %v, want exactly one for alpha", clicks)
	}
}

func TestStaysOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	if got := call(h, "GET", "/v1/trips/"+trip.Id+"/stays/links", "owner", "").Code; got != 503 {
		t.Fatalf("got %d, want 503", got)
	}
}
