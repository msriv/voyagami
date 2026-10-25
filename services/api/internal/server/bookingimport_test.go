package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/importer"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeParser struct {
	got   importer.Input
	cands []importer.Candidate
	err   error
}

func (f *fakeParser) Parse(_ context.Context, in importer.Input) ([]importer.Candidate, error) {
	f.got = in
	return f.cands, f.err
}

func TestImportBookings(t *testing.T) {
	trips := newFakeTrips()
	parser := &fakeParser{cands: []importer.Candidate{{
		Confidence: 0.9,
		Booking: model.Booking{
			Kind: "flight", Title: "BLR to HAN", Provider: "VietJet",
			StartsAt: time.Date(2026, 11, 19, 4, 0, 0, 0, time.UTC),
			From:     &model.BookingPlace{Code: "BLR", Name: "Bengaluru"},
			Price:    &model.Money{AmountMinor: 1234500, Currency: "INR"},
		},
	}}}
	v := tokenVerifier{"owner": "u-owner", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithBookings(&fakeBookings{byID: map[string]model.Booking{}}), WithImporter(parser))
	var trip api.Trip
	if err := json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip); err != nil {
		t.Fatal(err)
	}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	path := "/v1/trips/" + trip.Id + "/bookings/import"
	pdf := base64.StdEncoding.EncodeToString([]byte("%PDF-1.4 test"))

	var res api.BookingImportResult
	rec := call(h, "POST", path, "owner", `{"text":"Your flight is confirmed"}`)
	if rec.Code != 200 {
		t.Fatalf("text: %d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if len(res.Candidates) != 1 || res.Candidates[0].Booking.Title != "BLR to HAN" || res.Candidates[0].Confidence != 0.9 ||
		res.Candidates[0].Booking.Price == nil || parser.got.Text != "Your flight is confirmed" {
		t.Errorf("text result: %+v / %+v", res, parser.got)
	}
	if rec := call(h, "POST", path, "owner", `{"pdfBase64":"`+pdf+`"}`); rec.Code != 200 || string(parser.got.PDF) != "%PDF-1.4 test" {
		t.Errorf("pdf: %d %s", rec.Code, rec.Body)
	}

	tests := []struct {
		name, token, body string
		want              int
	}{
		{"viewer cannot import", "viewer", `{"text":"x"}`, 403},
		{"non-member", "other", `{"text":"x"}`, 404},
		{"signed out", "", `{"text":"x"}`, 401},
		{"nothing sent", "owner", `{}`, 400},
		{"both sent", "owner", `{"text":"x","pdfBase64":"` + pdf + `"}`, 400},
		{"text too long", "owner", `{"text":"` + strings.Repeat("a", importer.MaxTextRunes+1) + `"}`, 400},
		{"not base64", "owner", `{"pdfBase64":"!!!"}`, 400},
		{"not a pdf", "owner", `{"pdfBase64":"` + base64.StdEncoding.EncodeToString([]byte("hello")) + `"}`, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := call(h, "POST", path, tt.token, tt.body).Code; got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}

	parser.cands, parser.err = nil, importer.ErrNothingFound
	rec = call(h, "POST", path, "owner", `{"text":"hello"}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"candidates":[]`) {
		t.Errorf("nothing found: %d %s", rec.Code, rec.Body)
	}
	parser.err = errors.New("model down")
	rec = call(h, "POST", path, "owner", `{"text":"secret words"}`)
	if rec.Code != 503 || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("model failure: %d %s", rec.Code, rec.Body)
	}
}

func TestImportBookingsOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()))
	if got := call(h, "POST", "/v1/trips/t/bookings/import", "owner", `{"text":"x"}`).Code; got != 503 {
		t.Fatalf("got %d, want 503", got)
	}
}
