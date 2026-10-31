package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/places"
)

type fakePlaces struct {
	near *places.Near
	err  error
}

func (f *fakePlaces) Search(_ context.Context, q string, near *places.Near) ([]places.Result, error) {
	f.near = near
	return []places.Result{{Name: q, Address: "Hanoi", Lat: 21.03, Lng: 105.85, Category: "food"}}, f.err
}

func TestSearchPlaces(t *testing.T) {
	v := tokenVerifier{"owner": "u-owner"}
	f := &fakePlaces{}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithPlaces(f))

	if rec := call(h, "GET", "/v1/places/search?q=pho", "", ""); rec.Code != 401 {
		t.Errorf("no token: %d", rec.Code)
	}
	rec := call(h, "GET", "/v1/places/search?q=pho&lat=21&lng=105", "owner", "")
	var out api.PlaceSearchResults
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != 200 || len(out.Results) != 1 || out.Results[0].Name != "pho" || f.near == nil || f.near.Lat != 21 {
		t.Fatalf("search: %d %s", rec.Code, rec.Body)
	}
	if rec := call(h, "GET", "/v1/places/search?q=p", "owner", ""); rec.Code != 400 {
		t.Errorf("short q: %d", rec.Code)
	}
	f.err = errors.New("boom")
	if rec := call(h, "GET", "/v1/places/search?q=pho", "owner", ""); rec.Code != 503 {
		t.Errorf("provider down: %d", rec.Code)
	}
	off := NewHandler(slog.New(slog.DiscardHandler), v)
	if rec := call(off, "GET", "/v1/places/search?q=pho", "owner", ""); rec.Code != 503 {
		t.Errorf("without provider: %d", rec.Code)
	}
}

func TestResolvePlaceLink(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"})
	post := func(tok, body string) int { return call(h, "POST", "/v1/places/resolve", tok, body).Code }
	if c := post("", `{"url":"geo:21,105"}`); c != 401 {
		t.Errorf("no token: %d", c)
	}
	rec := call(h, "POST", "/v1/places/resolve", "owner", `{"url":"https://www.google.com/maps/place/Pho+10/@21.03,105.85,15z"}`)
	var out api.PlaceSearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != 200 || out.Name != "Pho 10" || out.Lat != 21.03 {
		t.Fatalf("resolve: %d %s", rec.Code, rec.Body)
	}
	if c := post("owner", `{"url":"https://evil.example/@21,105"}`); c != 404 {
		t.Errorf("foreign link: %d", c)
	}
	if c := post("owner", `{}`); c != 400 && c != 404 {
		t.Errorf("empty: %d", c)
	}
}
