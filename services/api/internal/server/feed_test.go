package server

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/msriv/voyagami/services/api/internal/api"
	"github.com/msriv/voyagami/services/api/internal/model"
)

// fakeFeed hands out whatever the test sends on its channel.
type fakeFeed struct{ ch chan model.Change }

func (f *fakeFeed) Watch(ctx context.Context, _ string, emit func(model.Change)) error {
	for {
		select {
		case c := <-f.ch:
			emit(c)
		case <-ctx.Done():
			return nil
		}
	}
}

func TestChangeFeed(t *testing.T) {
	trips := newFakeTrips()
	feed := &fakeFeed{ch: make(chan model.Change)}
	v := tokenVerifier{"owner": "u-owner", "viewer": "u-viewer", "other": "u-other"}
	h := NewHandler(slog.New(slog.DiscardHandler), v, WithTrips(trips), WithFeed(feed))
	var trip api.Trip
	if err := json.Unmarshal(call(h, "POST", "/v1/trips", "owner", newTripBody).Body.Bytes(), &trip); err != nil {
		t.Fatal(err)
	}
	trips.members[trip.Id]["u-viewer"] = model.Member{UID: "u-viewer", Role: model.RoleViewer, Status: model.MemberActive}
	path := "/v1/trips/" + trip.Id + "/events"

	if got := call(h, "GET", path, "other", "").Code; got != 404 {
		t.Errorf("non-member: %d", got)
	}
	if got := call(h, "GET", path, "", "").Code; got != 401 {
		t.Errorf("signed out: %d", got)
	}

	srv := httptest.NewServer(h)
	defer srv.Close()
	open := func(lastID string) (*bufio.Scanner, func()) {
		ctx, cancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+path, nil)
		req.Header.Set("Authorization", "Bearer viewer")
		if lastID != "" {
			req.Header.Set("Last-Event-ID", lastID)
		}
		resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // closed by the returned func
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
			t.Fatalf("open: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		return bufio.NewScanner(resp.Body), func() { cancel(); _ = resp.Body.Close() }
	}
	// next returns the next event's name and data, skipping comments and ids.
	next := func(sc *bufio.Scanner) (event, data string) {
		for sc.Scan() {
			line := sc.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				return event, strings.TrimPrefix(line, "data: ")
			}
		}
		t.Fatal("stream ended")
		return
	}

	sc, closeStream := open("")
	feed.ch <- model.Change{Kind: "stops", ID: "s1", Op: "upsert"}
	ev, data := next(sc)
	if ev != "change" || data != `{"kind":"stops","id":"s1","op":"upsert"}` {
		t.Fatalf("got %q %q", ev, data)
	}
	closeStream()

	// A reconnect starts with a resync.
	sc, closeStream = open("7")
	defer closeStream()
	if ev, _ := next(sc); ev != "resync" {
		t.Fatalf("reconnect starts with %q", ev)
	}
}

func TestChangeFeedOffAnswers503(t *testing.T) {
	h := NewHandler(slog.New(slog.DiscardHandler), tokenVerifier{"owner": "u-owner"}, WithTrips(newFakeTrips()))
	if got := call(h, "GET", "/v1/trips/t/events", "owner", "").Code; got != 503 {
		t.Fatalf("got %d, want 503", got)
	}
}

func TestChangeStreamEndsAtMaxAge(t *testing.T) {
	rec := httptest.NewRecorder()
	r := streamResponse{ctx: context.Background(), s: changeStream{
		feed: &fakeFeed{ch: make(chan model.Change)}, maxAge: 50 * time.Millisecond, pulse: 10 * time.Millisecond,
	}}
	if err := r.VisitStreamTripChangesResponse(rec); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rec.Body.String(), ": keep-alive") {
		t.Fatalf("no heartbeat: %q", rec.Body)
	}
}
