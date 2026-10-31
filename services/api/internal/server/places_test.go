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
