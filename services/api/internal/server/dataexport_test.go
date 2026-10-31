package server

import (
	"archive/zip"
	"bytes"
	"log/slog"
	"testing"

	"github.com/msriv/voyagami/services/api/internal/export"
)

func TestExportMyData(t *testing.T) {
	store := newFakeTrips()
	v := tokenVerifier{"owner": "u-owner", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(store), WithExport(export.Sources{Trips: store}))
	call(h, "POST", "/v1/trips", "owner", newTripBody)

	rec := call(h, "GET", "/v1/me/export", "owner", "")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("%d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	if len(names) != 1 || names[0] != "trips.json" {
		t.Errorf("files %v", names)
	}

	// Someone with no trips still gets a ZIP, with an empty trip list.
	if c := call(h, "GET", "/v1/me/export", "other", "").Code; c != 200 {
		t.Errorf("no trips: %d", c)
	}
	if c := call(h, "GET", "/v1/me/export", "", "").Code; c != 401 {
		t.Errorf("needs sign-in: %d", c)
	}
	off := NewHandler(slog.New(slog.DiscardHandler), v)
	if c := call(off, "GET", "/v1/me/export", "owner", "").Code; c != 503 {
		t.Errorf("off: %d", c)
	}
}
