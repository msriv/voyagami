package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/flightstatus"
	"github.com/msriv/voyagami/services/api/internal/model"
)

type fakeFlights struct {
	got flightstatus.Flight
	err error
}

func (f *fakeFlights) Status(_ context.Context, fl flightstatus.Flight) (flightstatus.Status, error) {
	f.got = fl
	return flightstatus.Status{State: flightstatus.Delayed, Departure: flightstatus.Leg{Estimated: time.Date(2027, 2, 1, 3, 0, 0, 0, time.UTC), Gate: "B4"}}, f.err
}

func TestFlightStatus(t *testing.T) {
	trips := newFakeTrips()
	fl := &fakeFlights{}
	bk := &fakeBookings{byID: map[string]model.Booking{
		"f1": {ID: "f1", Kind: "flight", Title: "AI 302 Delhi to Hanoi", StartsAt: time.Date(2027, 2, 1, 2, 30, 0, 0, time.UTC)},
		"f2": {ID: "f2", Kind: "flight", Title: "Delhi to Hanoi"},
		"s1": {ID: "s1", Kind: "stay", Title: "Hotel 12"},
	}}
	v := tokenVerifier{"owner": "u-owner", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithBookings(bk), WithFlightStatus(fl))
	var trip api.Trip
	_ = json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip)
	path := "/v1/trips/" + trip.Id + "/bookings/"

	rec := call(h, "GET", path+"f1/flight-status", "owner", "")
	var s api.FlightStatus
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	if rec.Code != 200 || s.State != "delayed" || s.Departure == nil || *s.Departure.Gate != "B4" || fl.got.Carrier != "AI" || fl.got.Number != "302" {
		t.Fatalf("%d %s, asked %+v", rec.Code, rec.Body, fl.got)
	}
	for name, p := range map[string]string{"no flight number": "f2", "not a flight": "s1", "missing": "nope"} {
		if c := call(h, "GET", path+p+"/flight-status", "owner", "").Code; c != 404 {
			t.Errorf("%s: %d", name, c)
		}
	}
	if c := call(h, "GET", path+"f1/flight-status", "other", "").Code; c != 404 {
		t.Errorf("non-member: %d", c)
	}
	fl.err = errors.New("down")
	if c := call(h, "GET", path+"f1/flight-status", "owner", "").Code; c != 503 {
		t.Errorf("provider down: %d", c)
	}
}

func TestFlightStatusOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()), WithBookings(&fakeBookings{byID: map[string]model.Booking{}}))
	if c := call(h, "GET", "/v1/trips/t/bookings/b/flight-status", "owner", "").Code; c != 503 {
		t.Errorf("%d", c)
	}
}
